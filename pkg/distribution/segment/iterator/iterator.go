package iterator

import (
	"context"
	"fmt"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/proxy"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/query"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
	"time"
)

type Iterator interface {
	// Source emits values from the iterator to a channel (confluence.Stream).
	// To bind a stream, call Iterator.OutTo(stream). The iterator will then send
	// all values to it. Iterator.OutTo must be called before any iteration methods,
	// or else values will be sent to a nil stream.
	confluence.Source[Response]
	// Next retrieves the next segment of each channel's data.
	// Returns true if the current Iterator.View is pointing to any valid segments.
	// It's important to note that if channel data is non-contiguous, calls to Next
	// may return segments that occupy different ranges of time.
	Next() bool
	// Prev retrieves the previous segment of each channel's data.
	// Returns true if the current Iterator.View is pointing to any valid segments.
	// It's important to note that if channel data is non-contiguous, calls to Prev
	// may return segments that occupy different ranges of time.
	Prev() bool
	// First returns the first segment of each channel's data.
	// Returns true if the current Iterator.View is pointing to any valid segments.
	// It's important to note that if channel data is non-contiguous, calls to First
	// may return segments that occupy different ranges of time.
	First() bool
	// Last returns the last segment of each channel's data.
	// Returns true if the current Iterator.View is pointing to any valid segments.
	// It's important to note that if channel data is non-contiguous, calls to Last
	// may return segments that occupy different ranges of time.
	Last() bool
	// NextSpan reads all channel data occupying the next span of time. Returns true
	// if the current Iterator.View is pointing to any valid segments.
	NextSpan(span telem.TimeSpan) bool
	// PrevSpan reads all channel data occupying the previous span of time. Returns true
	// if the current Iterator.View is pointing to any valid segments.
	PrevSpan(span telem.TimeSpan) bool
	// NextRange seeks the Iterator to the start of the range and reads all channel data
	// until the end of the range.
	NextRange(tr telem.TimeRange) bool
	// SeekFirst seeks the iterator the start of the iterator range.
	// Returns true if the current Iterator.View is pointing to any valid segments.
	SeekFirst() bool
	// SeekLast seeks the iterator the end of the iterator range.
	// Returns true if the current Iterator.View is pointing to any valid segments.
	SeekLast() bool
	// SeekLT seeks the iterator to the first whose timestamp is less than or equal
	// to the given timestamp. Returns true if the current Iterator.View is pointing
	// to any valid segments.
	SeekLT(t telem.TimeStamp) bool
	// SeekGE seeks the iterator to the first whose timestamp is greater than the
	// given timestamp. Returns true if the current Iterator.View is pointing to
	// any valid segments.
	SeekGE(t telem.TimeStamp) bool
	// Close closes the Iterator, ensuring that all in-progress reads complete
	// before closing the Source outlet. All iterators must be Closed, or the
	// distribution layer will panic.
	Close() error
	Valid() bool
	Error() error
	Exhaust() bool
}

func New(
	db cesium.DB,
	svc *channel.Service,
	resolver aspen.HostResolver,
	tran Transport,
	rng telem.TimeRange,
	keys channel.Keys,
) (Iterator, error) {
	ctx, cancel := signal.Background()

	// First we need to check if all the channels exists and are retrievable in the
	// database.
	if err := validateChannelKeys(ctx, svc, keys); err != nil {
		cancel()
		return nil, err
	}

	// Next we determine IDs of all the target nodes we need to open iterators on.
	batch := proxy.NewBatchFactory[channel.Key](resolver.HostID()).Batch(keys)

	var (
		needRemote        = len(batch.Remote) > 0
		needLocal         = len(batch.Local) > 0
		requests          = confluence.NewPipeline[Request]()
		responses         = confluence.NewPipeline[Response]()
		numSenders        = len(keys.Nodes())
		numReceivers      = 0
		receiverAddresses []address.Address
	)

	if needRemote {
		numSenders += 1

		sender, receivers, err := openRemoteIterators(ctx, tran, batch.Remote, rng, resolver)
		if err != nil {
			cancel()
			return nil, err
		}

		// Set up our sender as a sink for the request pipeline.
		requests.Sink("sender", sender)

		// Set up our remote receivers as sources for the response pipeline.
		receiverAddresses = make([]address.Address, len(receivers))
		for i, c := range receivers {
			addr := address.Address(fmt.Sprintf("client-%v", i+1))
			receiverAddresses[i] = addr
			responses.Source(addr, c)
		}
	}

	if needLocal {
		numSenders += 1

		localIter, err := newLocalIterator(db, resolver.HostID(), rng, batch.Local)
		if err != nil {
			cancel()
			return nil, err
		}

		addr := address.Address("local")

		// Set up our local iterator as a sink for the request pipeline.
		requests.Sink(addr, localIter)
		// And as a source for the response pipeline.
		responses.Source(addr, localIter)

		receiverAddresses = append(receiverAddresses, addr)
	}

	// The synchronizer checks that all nodes have acknowledged an iteration
	// request. This is used to return ok = true from the iterator methods.
	sync := &synchronizer{nodeIDs: keys.Nodes(), timeout: 1 * time.Second}

	// Open a ackFilter that will route acknowledgement responses to the iterator
	// synchronizer. We expect an ack from each remote iterator as well as the
	// local iterator, so we set our buffer cap at numReceivers.
	syncMessages := confluence.NewStream[Response](numReceivers)
	sync.InFrom(syncMessages)

	// Send rejects from the ackFilter to the synchronizer.
	filter := newAckRouter(syncMessages)
	responses.Sink("filter", filter)

	// emitter emits method calls as requests to stream.
	emit := &emitter{}
	requests.Source("emitter", emit)

	requestBuilder := requests.NewRouteBuilder()
	responseBuilder := responses.NewRouteBuilder()

	var routeEmitterTo address.Address

	// We need to configure different pipelines to optimize for particular cases.
	if needRemote && needLocal {
		// Open a broadcaster that will multiply requests to both the local and remote
		// iterators.
		requests.Sink("broadcaster", &confluence.Confluence[Request]{})
		routeEmitterTo = "broadcaster"

		// We use confluence.StitchWeave here to dedicate a channel to both the
		// sender and local, so that they both receive a copy of the emitted request.
		requestBuilder.Route(confluence.MultiRouter[Request]{
			SourceTargets: []address.Address{"broadcaster"},
			SinkTargets:   []address.Address{"sender", "local"},
			Capacity:      1,
			Stitch:        confluence.StitchWeave,
		})
	} else if needRemote {
		// If we only have remote iterators, we can skip the broadcasting step
		// and forward requests from the emitter directly to the sender.
		routeEmitterTo = "sender"
	} else {
		// If we only have local iterators, we can skip the broadcasting step
		// and forward requests from the emitter directly to the local iterator.
		routeEmitterTo = "local"
	}

	requestBuilder.RouteUnary("emitter", routeEmitterTo, 0)

	// Route all responses from our receivers to the ackFilter. Using a single channel
	// to link all the receivers to the ackFilter with a buffer capacity allowing
	// for 1 response per receiver at a time.
	responseBuilder.Route(confluence.MultiRouter[Response]{
		SourceTargets: receiverAddresses,
		SinkTargets:   []address.Address{"filter"},
		Stitch:        confluence.StitchUnary,
		Capacity:      numReceivers,
	})

	responseBuilder.PanicIfErr()
	requestBuilder.PanicIfErr()

	responses.Flow(ctx)
	requests.Flow(ctx)

	return &iterator{
		emitter:   emit,
		sync:      sync,
		wg:        ctx,
		shutdown:  cancel,
		ackFilter: filter,
	}, nil
}

type iterator struct {
	emitter  *emitter
	sync     *synchronizer
	shutdown context.CancelFunc
	wg       signal.WaitGroup
	_error   error
	*ackFilter
}

func (i *iterator) Next() bool {
	i.emitter.next()
	return i.ack(Next)
}

func (i *iterator) Prev() bool {
	i.emitter.Prev()
	return i.ack(Prev)
}

func (i *iterator) First() bool {
	i.emitter.First()
	return i.ack(First)
}

func (i *iterator) Last() bool {
	i.emitter.Last()
	return i.ack(Last)
}

func (i *iterator) NextSpan(span telem.TimeSpan) bool {
	i.emitter.NextSpan(span)
	return i.ack(NextSpan)
}

func (i *iterator) PrevSpan(span telem.TimeSpan) bool {
	i.emitter.PrevSpan(span)
	return i.ack(PrevSpan)
}

func (i *iterator) NextRange(tr telem.TimeRange) bool {
	i.emitter.NextRange(tr)
	return i.ack(NextRange)
}

func (i *iterator) SeekFirst() bool {
	i.emitter.SeekFirst()
	return i.ack(SeekFirst)
}

func (i *iterator) SeekLast() bool {
	i.emitter.SeekLast()
	return i.ack(SeekLast)
}

func (i *iterator) SeekLT(stamp telem.TimeStamp) bool {
	i.emitter.SeekLT(stamp)
	return i.ack(SeekLT)
}

func (i *iterator) SeekGE(stamp telem.TimeStamp) bool {
	i.emitter.SeekGE(stamp)
	return i.ack(SeekGE)
}

func (i *iterator) Exhaust() bool {
	i.emitter.Exhaust()
	return i.ack(Exhaust)
}

func (i *iterator) Valid() bool {
	i.emitter.Valid()
	return i.ack(Valid) && i.error() == nil
}

func (i *iterator) error() error {
	if i._error != nil {
		return i._error
	}
	if i.wg.AnyExited() {
		if err := i.wg.WaitOnAny(true); err != nil {
			i._error = err
		}
	}
	return nil
}

func (i *iterator) Error() error {
	if i.error() != nil {
		return i.error()
	}
	i.emitter.Error()
	ok, err := i.ackWithErr(Error)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("[iterator] - non positive ack")
	}
	return nil
}

func (i *iterator) ack(cmd Command) bool {
	ok, _ := i.ackWithErr(cmd)
	return ok
}

func (i *iterator) ackWithErr(cmd Command) (bool, error) {
	return i.sync.sync(context.Background(), cmd)
}

func (i *iterator) Close() error {

	// Wait for all iterator internal operations to complete.
	i.emitter.Close()

	// Wait for all nodes to acknowledge a safe closure.
	closeOk := i.ack(Close)

	// Shutdown iterator operations.
	i.shutdown()

	// Wait on all goroutines to exit.
	if err := i.wg.WaitOnAll(); err != nil && err != context.Canceled {
		return err
	}

	// If we received a negative ack with no error response, it probably means
	// we couldn't reach a node.
	if !closeOk {
		return errors.New("[segment.Iterator] - negative ack on close. node probably unreachable")
	}
	close(i.Filter.Out.Inlet())
	return nil
}

func validateChannelKeys(ctx context.Context, svc *channel.Service, keys []channel.Key) error {
	if len(keys) == 0 {
		return errors.New("[segment.iterator] - no channels provided to iterator")
	}
	exists, err := svc.NewRetrieve().WhereKeys(keys...).Exists(ctx)
	if !exists {
		return errors.Wrap(query.NotFound, "[segment.iterator] - channel keys not found")
	}
	if err != nil {
		return errors.Wrap(err, "[segment.iterator] - failed to validate channel keys")
	}
	return nil
}
