package writer

import (
	"context"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/proxy"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/arya-analytics/x/signal"
)

type Writer interface {
	Requests() chan<- Request
	Responses() <-chan Response
	Close() error
}

type writer struct {
	requests  chan<- Request
	responses <-chan Response
	wg        signal.WaitGroup
}

func (w *writer) Requests() chan<- Request { return w.requests }

func (w *writer) Responses() <-chan Response { return w.responses }

func (w *writer) Close() error { return w.wg.Wait() }

func New(
	ctx context.Context,
	db cesium.DB,
	svc *channel.Service,
	resolver aspen.HostResolver,
	tran Transport,
	keys channel.Keys,
) (Writer, error) {
	sCtx, cancel := signal.WithCancel(ctx)

	// First we need to check if all the channels exist and are retrievable in the
	//database.
	if err := core.ValidateChannelKeys(sCtx, svc, keys); err != nil {
		cancel()
		return nil, err
	}

	// TraverseTo we determine the IDs of all the target nodes we need to write to.
	batch := proxy.NewBatchFactory[channel.Key](resolver.HostID()).Batch(keys)

	var (
		pipe              = plumber.New()
		needRemote        = len(batch.Remote) > 0
		needLocal         = len(batch.Local) > 0
		receiverAddresses []address.Address
	)

	if needRemote {
		sender, receivers, err := openRemoteWriters(sCtx, tran, batch.Remote, resolver)
		if err != nil {
			cancel()
			return nil, err
		}

		// Set up our sender as a sink for the request pipeline.
		plumber.SetSink[Request](pipe, "remote", sender)

		// Set up our remote receivers as sources for the response pipeline.
		receiverAddresses = make([]address.Address, 0, len(receivers))
		for i, receiver := range receivers {
			addr := address.Newf("receiver-%d", i)
			receiverAddresses = append(receiverAddresses, addr)
			plumber.SetSource[Response](pipe, addr, receiver)
		}
	}

	if needLocal {
		w, err := newLocalWriter(sCtx, db, keys)
		if err != nil {
			cancel()
			return nil, err
		}
		addr := address.Address("local")
		plumber.SetSegment[Request, Response](pipe, addr, w)
		receiverAddresses = append(receiverAddresses, addr)
	}

	var routeRequestsTo address.Address

	if needRemote && needLocal {
		rls := newRemoteLocalSwitch(resolver.HostID())
		plumber.SetSegment[Request, Request](pipe, "remoteLocalSwitch", rls)
		routeRequestsTo = "remoteLocalSwitch"

		if err := (plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{"remoteLocalSwitch"},
			SinkTargets:   []address.Address{"remote", "local"},
			Stitch:        plumber.StitchWeave,
		}).Route(pipe); err != nil {
			panic(err)
		}
	} else if needRemote {
		routeRequestsTo = "remote"
	} else {
		routeRequestsTo = "local"
	}

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	if err := seg.RouteInletTo(routeRequestsTo); err != nil {
		panic(err)
	}
	if err := seg.RouteOutletFrom(receiverAddresses...); err != nil {
		panic(err)
	}

	input := confluence.NewStream[Request](0)
	output := confluence.NewStream[Response](0)
	seg.InFrom(input)
	seg.OutTo(output)

	seg.Flow(sCtx, confluence.CloseInletsOnExit())

	return &writer{responses: output.Outlet(), requests: input.Inlet(), wg: sCtx}, nil
}
