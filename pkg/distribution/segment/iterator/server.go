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

func newServer(db cesium.DB, transport Transport) *server {
	sf := &server{db: db}
	transport.Handle(sf.Handle)
	return sf
}

// Handle handles incoming requests from the transport.
func (sf *server) Handle(_ctx context.Context, server Server) error {
	ctx, cancel := signal.WithCancel(_ctx)
	defer cancel()

	// Block until we receive the first request from the remoteIterator. This message should
	// have an Open command that provides context for opening the cesium iterator.
	req, err := server.Receive()
	if err != nil {
		return err
	}
	if req.Command != Open {
		return errors.New("[segment.iterator] - server expected Open command")
	}

	// receiver receives requests from the server and pipes them into the
	// requestPipeline.
	receiver := &confluence.Receiver[Request]{Receiver: server}

	// sender receives responses from the response pipeline and sends
	// them over the network.
	sender := &confluence.Sender[Response]{Sender: server}

	iter, err := newLocalIterator(sf.db, sf.host, req.Range, req.Keys)
	if err != nil {
		return errors.Wrap(
			err,
			"[segment.iterator] - server failed to open cesium iterator",
		)
	}

	requests := confluence.NewStream[Request](0)
	responses := confluence.NewStream[Response](0)
	sender.InFrom(responses)
	iter.OutTo(responses)
	receiver.OutTo(requests)
	iter.InFrom(requests)

	receiver.Flow(ctx)
	sender.Flow(ctx)

	return ctx.WaitOnAny(true)
}
