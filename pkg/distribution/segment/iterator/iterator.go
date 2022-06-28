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
	confluence.Source[Response]
	Next() bool
	Prev() bool
	First() bool
	Last() bool
	NextSpan(span telem.TimeSpan) bool
	PrevSpan(span telem.TimeSpan) bool
	NextRange(tr telem.TimeRange) bool
	SeekFirst() bool
	SeekLast() bool
	SeekLT(t telem.TimeStamp) bool
	SeekGE(t telem.TimeStamp) bool
	Exhaust()
	Close() error
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

	requestPipeline := confluence.NewPipeline[Request]()
	responsePipeline := confluence.NewPipeline[Response]()

	clientAddresses := make([]address.Address, len(clients))
	for i, c := range clients {
		addr := address.Address(fmt.Sprintf("c-%d", i+1))
		clientAddresses[i] = addr
		requestPipeline.Sink(addr, c)
		responsePipeline.Source(addr, c)
	}

	// synchronizes iterator acknowledgements from all target node. If a response
	// from a target node is not received within the timeout, the iterator will
	// return false.
	sync := &synchronizer{nodeIDs: keys.Nodes(), timeout: 5 * time.Second}
	syncMessages := confluence.NewStream[Response](10)
	filter := newResponseFilter(syncMessages)
	sync.InFrom(syncMessages)

	responsePipeline.Segment("filter", filter)

	// emits iterator method calls as requests to the stream.
	emit := &emitter{}
	requestPipeline.Source("emitter", emit)

	// broadcast broadcasts requests to all target nodes.
	broadcast := &requestBroadcaster{}
	requestPipeline.Segment("broadcast", broadcast)

	iter := &iterator{
		emit:           emit,
		sync:           sync,
		wg:             ctx,
		shutdown:       cancel,
		responseFilter: filter,
	}

	responseBuilder := responsePipeline.NewRouteBuilder()
	requestBuilder := requestPipeline.NewRouteBuilder()

	requestBuilder.RouteUnary("emitter", "broadcast", 10)
	requestBuilder.Route(confluence.MultiRouter[Request]{
		FromAddresses: []address.Address{"broadcast"},
		ToAddresses:   clientAddresses,
		Capacity:      len(clientAddresses) + 5,
	})

	responseBuilder.Route(confluence.MultiRouter[Response]{
		FromAddresses: clientAddresses,
		ToAddresses:   []address.Address{"filter"},
		Stitch:        confluence.StitchUnary,
		Capacity:      len(clientAddresses) + 5,
	})

	requestBuilder.PanicIfErr()
	responseBuilder.PanicIfErr()

	responsePipeline.Flow(ctx)
	requestPipeline.Flow(ctx)

	signal.IterTransient(ctx, func(err error) {
		syncMessages.Inlet() <- Response{Error: err, Variant: ResponseVariantData}
	})

	return iter, nil
}

type iterator struct {
	emit     *emitter
	sync     *synchronizer
	shutdown context.CancelFunc
	wg       signal.WaitGroup
	*responseFilter
}

func (i *iterator) ack(cmd Command) bool { return i.sync.sync(context.Background(), cmd) }

func (i *iterator) ackRes(cmd Command) ([]Response, bool) {
	return i.sync.syncWithRes(context.Background(), cmd)
}

func (i *iterator) Next() bool { i.emit.next(); return i.ack(Next) }

func (i *iterator) Prev() bool { i.emit.Prev(); return i.ack(Prev) }

func (i *iterator) First() bool { i.emit.First(); return i.ack(First) }

func (i *iterator) Last() bool { i.emit.Last(); return i.ack(Last) }

func (i *iterator) NextSpan(span telem.TimeSpan) bool { i.emit.NextSpan(span); return i.ack(NextSpan) }

func (i *iterator) PrevSpan(span telem.TimeSpan) bool {
	i.emit.PrevSpan(
		span)
	return i.ack(PrevSpan)
}

func (i *iterator) NextRange(tr telem.TimeRange) bool {
	i.emit.NextRange(
		tr)
	return i.ack(NextRange)
}

func (i *iterator) SeekFirst() bool { i.emit.SeekFirst(); return i.ack(SeekFirst) }

func (i *iterator) SeekLast() bool { i.emit.SeekLast(); return i.ack(SeekLast) }

func (i *iterator) SeekLT(stamp telem.TimeStamp) bool {
	i.emit.SeekLT(
		stamp)
	return i.ack(SeekLT)
}

func (i *iterator) SeekGE(stamp telem.TimeStamp) bool {
	i.emit.SeekGE(stamp)
	return i.ack(SeekGE)
}

func (i *iterator) Close() error {
	// Wait for all iterator internal operations to complete.
	i.emit.Close()

	// Wait for all nodes to acknowledge a safe closure.
	responses, closeOk := i.ackRes(Close)

	// Wait for all nodes to finish transmitting their last value.
	eofOk := i.ack(EOF)

	// Shutdown iterator operations.
	i.shutdown()

	// Wait on all goroutines to complete.
	if err := i.wg.WaitOnAll(); err != nil && err != context.Canceled {
		return err
	}

	// Check responses for errors.
	for _, res := range responses {
		if res.Error != nil {
			return res.Error
		}
	}

	// If we received a negative ack with no error response, it probably means
	// we couldn't reach a node.
	if !closeOk || !eofOk {
		return errors.New(
			"[segment.Iterator] - received a non-positive ack on close. node probably unreachable.",
		)
	}

	i.Filter.Out.Close()

	return nil
}

func (i *iterator) Exhaust() { i.emit.Exhaust() }

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
