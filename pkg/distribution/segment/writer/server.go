package writer

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/arya-analytics/x/confluence/transfluence"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/transport"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
)

type server struct {
	host   node.ID
	db     cesium.DB
	logger *zap.SugaredLogger
}

func NewServer(db cesium.DB, host node.ID, transport Transport) *server {
	sf := &server{db: db}
	transport.Handle(sf.Handle)
	return sf
}

func (sf *server) Handle(_ctx context.Context, server Server) error {
	ctx, cancel := signal.WithCancel(_ctx)
	defer func() {
		cancel()
		if err := ctx.Wait(); err != nil && err != context.
			Canceled && err != context.DeadlineExceeded {
			sf.logger.Error(err)
		}
	}()

	// Block until we receive the first request from the remote writer. This message
	// should have an OpenKeys command that provides context for opening the cesium
	// writer.
	req, err := server.Receive()
	if err != nil {
		return err
	}
	if len(req.OpenKeys) == 0 {
		return errors.New("[segment.writer] - server expected OpenKeys to be defined")
	}

	receiver := &transfluence.Receiver[Request]{Receiver: server}
	sender := &transfluence.Sender[Response]{
		Sender: transport.SenderEmptyCloser[Response]{StreamSender: server},
	}

	writer := newLocalWriter(sf.db, req.OpenKeys)
	if err != nil {
		return errors.Wrap(err, "[segment.writer] - failed to open cesium writer")
	}

	pipe := plumber.New()
	plumber.SetSegment[Request, Response](pipe, "writer", writer)
	plumber.SetSource[Request](pipe, "receiver", receiver)
	plumber.SetSink[Response](pipe, "sender", sender)

	c := errutil.NewCatchSimple()

	c.Exec(plumber.UnaryRouter[Request]{
		SourceTarget: "receiver",
		SinkTarget:   "writer",
	}.PreRoute(pipe))

	c.Exec(plumber.UnaryRouter[Request]{
		SourceTarget: "writer",
		SinkTarget:   "sender",
	}.PreRoute(pipe))

	if c.Error() != nil {
		panic(c.Error())
	}

	pipe.Flow(ctx)

	return ctx.Wait()

}
