package iterator

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/telem"
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

	ctx := confluence.WrapContext().WithCtx(_ctx)

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

type streamIterator struct {
	confluence.Sink[Request]
	confluence.Source[Response]
}

func newStreamIterator(
	ctx context.Context,
	db cesium.DB,
	rng telem.TimeRange,
	keys channel.Keys,
) (*streamIterator, error) {
	_ctx := confluence.WrapContext().WithCtx(ctx)

	iter, err := db.NewRetrieve().WhereChannels(keys.Cesium()...).Iterate(_ctx.Ctx)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"[segment.iterator.serve] - failed to open cesium iterator",
		)
	}

	// cesiumPipeline moves cesium responses from the iterator source
	// to a point where they can be translated into a transport Response.
	cesiumPipeline := confluence.NewPipeline[cesium.RetrieveResponse]()

	requestPipeline := confluence.NewPipeline[Request]()
	responsePipeline := confluence.NewPipeline[Response]()

	cesiumPipeline.Source("iterator", iter)

	// executor executes requests as method calls on the iterator. Pipes
	// synchronous acknowledgements out to the response pipeline.
	te := newServerExecutor(iter)
	requestPipeline.Sink("executor", te)
	responsePipeline.Source("executor", te)

	// translator translates cesium responses from the iterator source into
	// responses transportable over the network.
	ts := newServerTranslator(keys.CesiumMap())
	cesiumPipeline.Sink("translator", ts)
	responsePipeline.Source("translator", ts)

	defer func() {
		if err := _ctx.Shutdown.Shutdown(); err != nil {
			panic(err)
		}
	}()

	requestBuilder := requestPipeline.NewRouteBuilder()
	requestBuilder.RouteUnary("receiver", "executor", 1)

	if requestBuilder.Error() != nil {
		return nil, requestBuilder.Error()
	}

	responseBuilder := responsePipeline.NewRouteBuilder()
	responseBuilder.RouteUnary("translator", "sender", 1)
	responseBuilder.RouteUnary("executor", "sender", 1)

	if responseBuilder.Error() != nil {
		return nil, responseBuilder.Error()
	}

	cesiumBuilder := cesiumPipeline.NewRouteBuilder()
	cesiumBuilder.RouteUnary("iterator", "translator", 1)

	if cesiumBuilder.Error() != nil {
		return nil, cesiumBuilder.Error()
	}

	requestPipeline.Flow(_ctx)
	responsePipeline.Flow(_ctx)
	requestPipeline.Flow(_ctx)

	streamIter := &streamIterator{}
	streamIter.Sink = requestPipeline
	streamIter.Source = responsePipeline

	return <-ctx.ErrC
}

type serverExecutor struct {
	iter cesium.StreamIterator
	confluence.CoreTranslator[Request, Response]
}

func newServerExecutor(iter cesium.StreamIterator) *serverExecutor {
	te := &serverExecutor{iter: iter}
	te.CoreTranslator.Translate = te.execute
	return te
}

func (te *serverExecutor) execute(ctx confluence.Context, req Request) Response {
	return executeRequest(te.iter, req)
}

type serverTranslator struct {
	wrapper *core.CesiumWrapper
	confluence.CoreTranslator[cesium.RetrieveResponse, Response]
}

func newServerTranslator(keyMap map[cesium.ChannelKey]channel.Key) *serverTranslator {
	wrapper := &core.CesiumWrapper{KeyMap: keyMap}
	ts := &serverTranslator{wrapper: wrapper}
	ts.CoreTranslator.Translate = ts.translate
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
