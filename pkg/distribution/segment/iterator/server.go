package iterator

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
)

type server struct {
	host node.ID
	db   cesium.DB
}

func NewServer(db cesium.DB, host node.ID, transport Transport) *server {
	sf := &server{db: db, host: host}
	transport.Handle(sf.Handle)
	return sf
}

// Handle handles incoming req from the transport.
func (sf *server) Handle(_ctx context.Context, server Server) error {
	ctx, cancel := signal.WithCancel(_ctx)
	defer func() {
		cancel()
		if err := ctx.WaitOnAll(); err != nil && err != context.
			Canceled && err != context.DeadlineExceeded {
			panic(err)
		}
	}()

	// Block until we receive the first request from the remoteIterator. This message should
	// have an Open command that provides context for opening the cesium iterator.
	req, err := server.Receive()
	if err != nil {
		return err
	}
	if req.Command != Open {
		return errors.New("[segment.iterator] - server expected Open command")
	}

	// receiver receives req from the server and pipes them into the
	// requestPipeline.
	receiver := confluence.GateSource[Request](&confluence.Receiver[Request]{Receiver: server})

	// sender receives res from the response pipeline and sends
	// them over the network.
	sender := confluence.GateSink[Response](&confluence.Sender[Response]{Sender: server})

	iter, err := newLocalIterator(sf.db, sf.host, req.Range, req.Keys)
	if err != nil {
		return errors.Wrap(err, "[segment.iterator] - cesium iterator failed to open")
	}

	requests := confluence.NewStream[Request](0)
	responses := confluence.NewStream[Response](0)
	sender.InFrom(responses)
	iter.OutTo(responses)
	receiver.OutTo(requests)
	iter.InFrom(requests)

	iter.Flow(ctx)
	receiver.Flow(ctx)
	sender.Flow(ctx)

	// The only way for hte iterator to exit is if the client closes the connection
	// or cancels the context. In both of these cases, the receiver will detect the
	// exit, so we just need to wait for it.
	return ctx.WaitOnAny(true)
}
