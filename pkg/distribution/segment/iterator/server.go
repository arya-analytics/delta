package iterator

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/confluence"
	"github.com/cockroachdb/errors"
)

type server struct {
	db cesium.DB
}

func newServer(transport Transport) *server {
	sf := new(server)
	transport.Handle(sf.Handle)
	return sf
}

// Handle handles incoming requests from the transport.
func (sf *server) Handle(_ctx context.Context, server Server) error {
	// Block until we receive the first request from the client. This message should
	// have an Open command that provides context for opening the cesium iterator.
	req, err := server.Receive()
	if err != nil {
		return err
	}
	if req.Command != Open {
		return errors.New("[segment.iterator.serve] - expected Open command")
	}

	// Open the cesium iterator.
	iter := sf.db.NewRetrieve().WhereChannels(req.Keys.Cesium()...).Iterate(_ctx)
	if iter.Error() != nil {
		return errors.Wrap(
			iter.Error(),
			"[segment.iterator.serve] - failed to open cesium iterator",
		)
	}

	ctx := confluence.NewContext().WithCtx(_ctx)

	// cesiumPipeline moves cesium responses from the iterator source
	// to a point where they can be translated into a transport Response.
	cesiumPipeline := confluence.NewPipeline[cesium.RetrieveResponse]()

	requestPipeline := confluence.NewPipeline[Request]()
	responsePipeline := confluence.NewPipeline[Response]()

	cesiumPipeline.Source("iterator", iter)

	// receiver receives requests from the server and pipes them into the
	// requestPipeline.
	receiver := &confluence.Receiver[Request]{Receiver: server}
	requestPipeline.Segment("receiver", receiver)

	// sender receives responses from the response pipeline and sends
	// them over the network.
	sender := &confluence.Sender[Response]{Sender: server}
	responsePipeline.Segment("sender", sender)

	// executor executes requests as method calls on the iterator. Pipes
	// synchronous acknowledgements out to the response pipeline.
	te := newServerExecutor(iter)
	requestPipeline.Sink("executor", te)
	responsePipeline.Source("executor", te)

	// translator translates cesium responses from the iterator source into
	// responses transportable over the network.
	ts := newServerTranslator(req.Keys.CesiumMap())
	cesiumPipeline.Sink("translator", ts)
	responsePipeline.Source("translator", ts)

	defer func() {
		if err := ctx.Shutdown.Shutdown(); err != nil {
			panic(err)
		}
	}()

	requestBuilder := requestPipeline.NewRouteBuilder()
	requestBuilder.RouteUnary("receiver", "executor", 1)

	if requestBuilder.Error() != nil {
		return requestBuilder.Error()
	}

	responseBuilder := responsePipeline.NewRouteBuilder()
	responseBuilder.RouteUnary("translator", "sender", 1)
	responseBuilder.RouteUnary("executor", "sender", 1)

	if responseBuilder.Error() != nil {
		return responseBuilder.Error()
	}

	cesiumBuilder := cesiumPipeline.NewRouteBuilder()
	cesiumBuilder.RouteUnary("iterator", "translator", 1)

	if cesiumBuilder.Error() != nil {
		return cesiumBuilder.Error()
	}

	requestPipeline.Flow(ctx)
	responsePipeline.Flow(ctx)
	requestPipeline.Flow(ctx)

	return <-ctx.ErrC
}

type serverExecutor struct {
	iter cesium.StreamIterator
	confluence.Translator[Request, Response]
}

func newServerExecutor(iter cesium.StreamIterator) *serverExecutor {
	te := &serverExecutor{iter: iter}
	te.Translator.Translate = te.execute
	return te
}

func (te *serverExecutor) execute(ctx confluence.Context, req Request) Response {
	return executeRequest(te.iter, req)
}

type serverTranslator struct {
	wrapper *core.CesiumWrapper
	confluence.Translator[cesium.RetrieveResponse, Response]
}

func newServerTranslator(keyMap map[cesium.ChannelKey]channel.Key) *serverTranslator {
	wrapper := &core.CesiumWrapper{KeyMap: keyMap}
	ts := &serverTranslator{wrapper: wrapper}
	ts.Translator.Translate = ts.translate
	return ts
}

func (te *serverTranslator) translate(
	ctx confluence.Context,
	req cesium.RetrieveResponse,
) Response {
	return Response{
		Variant:  ResponseVariantData,
		Error:    req.Error,
		Segments: te.wrapper.Wrap(req.Segments),
	}
}
