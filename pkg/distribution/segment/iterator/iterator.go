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
	transport Transport,
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

	var clients []confluence.Translator[Request, Response]

	if len(batch.Local) > 0 {
		localClient, err := newLocalIterator(db, resolver.HostID(), rng, batch.Local)
		if err != nil {
			cancel()
			return nil, err
		}
		clients = append(clients, localClient)
	}

	for targetNode, _keys := range batch.Remote {
		targetAddr, err := resolver.Resolve(targetNode)
		if err != nil {
			cancel()
			return nil, err
		}
		remoteIter, err := newRemoteIterator(ctx, transport, targetAddr, _keys, rng)
		if err != nil {
			cancel()
			return nil, err
		}
		clients = append(clients, remoteIter)
	}

	requests := confluence.NewPipeline[Request]()
	responses := confluence.NewPipeline[Response]()

	clientAddresses := make([]address.Address, len(clients))
	for i, c := range clients {
		addr := address.Address(fmt.Sprintf("client-%d", i+1))
		clientAddresses[i] = addr
		requests.Sink(addr, c)
		responses.Source(addr, c)
	}

	// synchronizes iterator acknowledgements from all target node. If a response
	// from a target node is not received within the timeout, the iterator will
	// return false.
	sync := &synchronizer{nodeIDs: keys.Nodes(), timeout: 5 * time.Second}
	syncMessages := confluence.NewStream[Response](len(clientAddresses))
	filter := newResponseFilter(syncMessages)
	sync.InFrom(syncMessages)
	responses.Segment("filter", filter)

	// emits iterator method calls as req to the stream.
	emit := &emitter{}
	requests.Source("emitter", emit)

	// broadcasts requests to all target nodes.
	broadcast := &requestBroadcaster{}
	requests.Segment("broadcast", broadcast)

	iter := &iterator{
		emit:           emit,
		sync:           sync,
		wg:             ctx,
		shutdown:       cancel,
		responseFilter: filter,
	}

	responseBuilder := responses.NewRouteBuilder()
	requestBuilder := requests.NewRouteBuilder()

	requestBuilder.RouteUnary("emitter", "broadcast", 10)
	requestBuilder.Route(confluence.MultiRouter[Request]{
		SourceTargets: []address.Address{"broadcast"},
		SinkTargets:   clientAddresses,
		Capacity:      len(clientAddresses) + 5,
	})

	responseBuilder.Route(confluence.MultiRouter[Response]{
		SourceTargets: clientAddresses,
		SinkTargets:   []address.Address{"filter"},
		Stitch:        confluence.StitchUnary,
		Capacity:      len(clientAddresses) + 5,
	})

	requestBuilder.PanicIfErr()
	responseBuilder.PanicIfErr()

	responses.Flow(ctx)
	requests.Flow(ctx)

	return iter, nil
}

type iterator struct {
	emit     *emitter
	sync     *synchronizer
	shutdown context.CancelFunc
	wg       signal.WaitGroup
	_error   error
	*responseFilter
}

func (i *iterator) Next() bool { i.emit.next(); return i.ack(Next) }

func (i *iterator) Prev() bool { i.emit.Prev(); return i.ack(Prev) }

func (i *iterator) First() bool { i.emit.First(); return i.ack(First) }

func (i *iterator) Last() bool { i.emit.Last(); return i.ack(Last) }

func (i *iterator) NextSpan(span telem.TimeSpan) bool { i.emit.NextSpan(span); return i.ack(NextSpan) }

func (i *iterator) PrevSpan(span telem.TimeSpan) bool { i.emit.PrevSpan(span); return i.ack(PrevSpan) }

func (i *iterator) NextRange(tr telem.TimeRange) bool { i.emit.NextRange(tr); return i.ack(NextRange) }

func (i *iterator) SeekFirst() bool { i.emit.SeekFirst(); return i.ack(SeekFirst) }

func (i *iterator) SeekLast() bool { i.emit.SeekLast(); return i.ack(SeekLast) }

func (i *iterator) SeekLT(stamp telem.TimeStamp) bool { i.emit.SeekLT(stamp); return i.ack(SeekLT) }

func (i *iterator) SeekGE(stamp telem.TimeStamp) bool { i.emit.SeekGE(stamp); return i.ack(SeekGE) }

func (i *iterator) Exhaust() bool { i.emit.Exhaust(); return i.ack(Exhaust) }

func (i *iterator) Valid() bool {
	i.emit.Valid()
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
	i.emit.Error()
	ok, err := i.ackWithErr(Error)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("[iterator] - non positive ack")
	}
	return nil
}

func (i *iterator) ack(cmd Command) bool { ok, _ := i.ackWithErr(cmd); return ok }

func (i *iterator) ackWithErr(cmd Command) (bool, error) {
	return i.sync.sync(context.Background(), cmd)
}

func (i *iterator) Close() error {

	// Wait for all iterator internal operations to complete.
	i.emit.Close()

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
