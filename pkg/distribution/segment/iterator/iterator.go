package iterator

import (
	"context"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/delta/pkg/proxy"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
	"time"
)

type Iterator interface {
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
	ctx context.Context,
	db cesium.DB,
	svc *channel.Service,
	resolver aspen.HostResolver,
	tran Transport,
	rng telem.TimeRange,
	keys channel.Keys,
	output chan<- Response,
) (Iterator, error) {
	sCtx, cancel := signal.WithCancel(ctx)

	// First we need to check if all the channels exist and are retrievable in the
	// database.
	if err := core.ValidateChannelKeys(ctx, svc, keys); err != nil {
		return nil, err
	}

	// Next we determine IDs of all the target nodes we need to open iterators on.
	batch := proxy.NewBatchFactory[channel.Key](resolver.HostID()).Batch(keys)

	var (
		pipe              = plumber.New()
		needRemote        = len(batch.Remote) > 0
		needLocal         = len(batch.Local) > 0
		numSenders        = 0
		numReceivers      = 0
		receiverAddresses []address.Address
	)

	if needRemote {
		numSenders += 1
		numReceivers += len(batch.Remote)

		sender, receivers, err := openRemoteIterators(sCtx, tran, batch.Remote, rng, resolver)
		if err != nil {
			cancel()
			return nil, err
		}

		// Set up our sender as a sink for the request pipeline.
		plumber.SetSink[Request](pipe, "sender", sender)

		// Set up our remote receivers as sources for the response pipeline.
		receiverAddresses = make([]address.Address, len(receivers))
		for i, c := range receivers {
			addr := address.Newf("client-%v", i+1)
			receiverAddresses[i] = addr
			plumber.SetSource[Response](pipe, addr, c)
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
		plumber.SetSegment[Request, Response](pipe, addr, localIter)
		receiverAddresses = append(receiverAddresses, addr)
	}

	// The synchronizer checks that all nodes have acknowledged an iteration
	// request. This is used to return ok = true from the iterator methods.
	sync := &synchronizer{nodeIDs: keys.Nodes(), timeout: 2 * time.Second}

	// Open a ackFilter that will route acknowledgement responses to the iterator
	// synchronizer. We expect an ack from each remote iterator as well as the
	// local iterator, so we set our buffer cap at numReceivers.
	syncMessages := confluence.NewStream[Response](numReceivers)
	sync.InFrom(syncMessages)

	// Send rejects from the ackFilter to the synchronizer.
	filter := newAckRouter(syncMessages)
	plumber.SetSegment[Response, Response](pipe, "filter", filter)

	// emitter emits method calls as requests to stream.
	emit := &emitter{}
	plumber.SetSource[Request](pipe, "emitter", emit)

	var (
		routeEmitterTo address.Address
		c              = errutil.NewCatchSimple()
	)

	// We need to configure different pipelines to optimize for particular cases.
	if needRemote && needLocal {
		// Open a broadcaster that will multiply requests to both the local and remote
		// iterators.
		plumber.SetSegment[Request, Request](
			pipe,
			"broadcaster",
			&confluence.DeltaMultiplier[Request]{},
		)
		routeEmitterTo = "broadcaster"

		// We use confluence.StitchWeave here to dedicate a channel to both the
		// sender and local, so that they both receive a copy of the emitted request.
		c.Exec(plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{"broadcaster"},
			SinkTargets:   []address.Address{"sender", "local"},
			Capacity:      1,
			Stitch:        plumber.StitchWeave,
		}.PreRoute(pipe))
	} else if needRemote {
		// If we only have remote iterators, we can skip the broadcasting step
		// and forward requests from the emitter directly to the sender.
		routeEmitterTo = "sender"
	} else {
		// If we only have local iterators, we can skip the broadcasting step
		// and forward requests from the emitter directly to the local iterator.
		routeEmitterTo = "local"
	}

	c.Exec(plumber.UnaryRouter[Request]{
		SourceTarget: "emitter",
		SinkTarget:   routeEmitterTo,
	}.PreRoute(pipe))

	c.Exec(plumber.MultiRouter[Response]{
		SourceTargets: receiverAddresses,
		SinkTargets:   []address.Address{"filter"},
		Stitch:        plumber.StitchUnary,
		Capacity:      1,
	}.PreRoute(pipe))

	if c.Error() != nil {
		panic(c.Error())
	}

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	if err := seg.RouteOutletFrom("filter"); err != nil {
		panic(err)
	}

	seg.OutTo(confluence.NewInlet[Response](output))
	seg.Flow(sCtx, confluence.CloseInletsOnExit())

	return &iterator{emitter: emit, sync: sync, wg: sCtx, cancel: cancel}, nil
}

type iterator struct {
	emitter *emitter
	sync    *synchronizer
	cancel  context.CancelFunc
	wg      signal.WaitGroup
	_error  error
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
		i.cancel()
		if err := i.wg.Wait(); err != nil {
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
	defer i.cancel()

	// Wait for all iterator internal operations to complete.
	i.emitter.Close()

	// Wait for all nodes to acknowledge a safe closure.
	if ok := i.ack(Close); !ok {
		return errors.New("[segment.iterator] - negative ack on close. node probably unreachable")
	}

	i.emitter.Out.Close()

	// Wait on all goroutines to exit.
	return i.wg.Wait()
}
