package iterator

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/confluence"
	"github.com/cockroachdb/errors"
)

type serverFactory struct {
	db cesium.DB
}

func newServerFactory(transport Transport) {
	sf := &serverFactory{}
	transport.Handle(sf.Handle)
}

func (sf *serverFactory) Handle(_ctx context.Context, _server Server) error {
	// Block until we receive the first request from the client. This message should
	// have an Open command that provides context for opening the cesium iterator.
	req, err := _server.Receive()
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

	// Set up the processing pipeline.

	cesiumPipeline := confluence.NewPipeline[cesium.RetrieveResponse]()
	cesiumPipeline.Source("iter", iter)

	ctx := confluence.DefaultContext()
	ctx.Ctx = _ctx

	requestPipeline := confluence.NewPipeline[Request]()
	responsePipeline := confluence.NewPipeline[Response]()

	receiver := &confluence.Receiver[Request]{Receiver: _server}
	sender := &confluence.Sender[Response]{Sender: _server}
	requestPipeline.Segment("receiver", receiver)
	responsePipeline.Segment("sender", sender)

	te := newTranslateExecutor(iter)
	requestPipeline.Sink("translator", te)
	responsePipeline.Source("translator", te)

	ts := newTranslateSender(req.Keys.CesiumMap())
	cesiumPipeline.Sink("translateSender", ts)
	responsePipeline.Source("translateSender", ts)

	defer func() {
		if err := ctx.Shutdown.Shutdown(); err != nil {
			panic(err)
		}
	}()

	requestPipeline.Flow(ctx)
	responsePipeline.Flow(ctx)
	requestPipeline.Flow(ctx)

	return <-ctx.ErrC
}

type translateExecutor struct {
	iter cesium.StreamIterator
	confluence.Translator[Request, Response]
}

func newTranslateExecutor(iter cesium.StreamIterator) *translateExecutor {
	te := &translateExecutor{iter: iter}
	te.Translator.Translate = te.execute
	return te
}

func (te *translateExecutor) execute(ctx confluence.Context, req Request) Response {
	return executeRequest(te.iter, req)
}

type translateSender struct {
	wrapper *core.CesiumWrapper
	confluence.Translator[cesium.RetrieveResponse, Response]
}

func newTranslateSender(keyMap map[cesium.ChannelKey]channel.Key) *translateSender {
	wrapper := &core.CesiumWrapper{KeyMap: keyMap}
	ts := &translateSender{wrapper: wrapper}
	ts.Translator.Translate = ts.translate
	return ts
}

func (te *translateSender) translate(ctx confluence.Context,
	req cesium.RetrieveResponse) Response {
	return Response{
		Variant:  ResponseVariantData,
		Error:    req.Error,
		Segments: te.wrapper.Wrap(req.Segments),
	}
}
