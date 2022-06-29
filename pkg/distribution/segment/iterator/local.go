package iterator

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
	"io"
)

type localIterator struct {
	requests        confluence.Sink[Request]
	responses       confluence.Source[Response]
	cesiumResponses confluence.Segment[cesium.RetrieveResponse]
}

func (s *localIterator) Flow(ctx signal.Context, opts ...confluence.FlowOption) {
	s.requests.Flow(ctx)
	s.responses.Flow(ctx)
	s.cesiumResponses.Flow(ctx)
}

func (s *localIterator) OutTo(inlets ...confluence.Inlet[Response]) { s.responses.OutTo(inlets...) }

func (s *localIterator) InFrom(outlets ...confluence.Outlet[Request]) { s.requests.InFrom(outlets...) }

func newLocalIterator(
	db cesium.DB,
	host node.ID,
	rng telem.TimeRange,
	keys channel.Keys,
) (confluence.Translator[Request, Response], error) {
	iter, err := db.NewRetrieve().
		WhereTimeRange(rng).
		WhereChannels(keys.Cesium()...).
		Iterate()
	if err != nil {
		return nil, errors.Wrap(
			err,
			"[segment.iterator.serve] - failed to open cesium iterator",
		)
	}

	// to a point where they can be translated into a transport Response.
	cesiumPipeline := confluence.NewPipeline[cesium.RetrieveResponse]()

	requestPipeline := confluence.NewPipeline[Request]()
	responsePipeline := confluence.NewPipeline[Response]()

	// cesiumPipeline receives segments from the iterator.
	cesiumPipeline.Source("iterator", iter)

	// executor executes requests as method calls on the iterator. Pipes
	// synchronous acknowledgements out to the response pipeline.
	te := newRequestExecutor(host, iter)
	requestPipeline.Sink("executor", te)
	responsePipeline.Source("executor", te)

	// translator translates cesium responses from the iterator source into
	// responses transportable over the network.
	ts := newCesiumResponseTranslator(keys.CesiumMap())
	cesiumPipeline.Sink("translator", ts)
	responsePipeline.Source("translator", ts)

	requestBuilder := requestPipeline.NewRouteBuilder()
	responseBuilder := responsePipeline.NewRouteBuilder()

	cesiumBuilder := cesiumPipeline.NewRouteBuilder()
	cesiumBuilder.RouteUnary("iterator", "translator", 10)

	requestBuilder.RouteInletTo("executor")
	// Both our executor and translator return responses.
	responseBuilder.RouteOutletFrom("executor", "translator")

	requestBuilder.PanicIfErr()
	responseBuilder.PanicIfErr()
	cesiumBuilder.PanicIfErr()

	return &localIterator{
		requests:        requestPipeline,
		responses:       responsePipeline,
		cesiumResponses: cesiumPipeline,
	}, nil
}

type requestExecutor struct {
	host node.ID
	iter cesium.StreamIterator
	confluence.CoreTranslator[Request, Response]
}

func newRequestExecutor(host node.ID, iter cesium.StreamIterator) confluence.Translator[Request, Response] {
	te := &requestExecutor{iter: iter, host: host}
	te.CoreTranslator.Translate = te.execute
	return confluence.GateTranslator[Request, Response](te)
}

func (te *requestExecutor) execute(ctx signal.Context, req Request) (Response, bool, error) {
	res := executeRequest(ctx, te.host, te.iter, req)
	// If we don't have a valid response, don't send it.
	return res, res.Variant != 0, nil
}

type cesiumResponseTranslator struct {
	wrapper *core.CesiumWrapper
	confluence.CoreTranslator[cesium.RetrieveResponse, Response]
}

func newCesiumResponseTranslator(keyMap map[cesium.ChannelKey]channel.
Key) confluence.Translator[cesium.RetrieveResponse, Response] {
	wrapper := &core.CesiumWrapper{KeyMap: keyMap}
	ts := &cesiumResponseTranslator{wrapper: wrapper}
	ts.CoreTranslator.Translate = ts.translate
	return confluence.GateTranslator[cesium.RetrieveResponse, Response](ts)
}

func (te *cesiumResponseTranslator) translate(
	ctx signal.Context,
	res cesium.RetrieveResponse,
) (Response, bool, error) {
	if errors.Is(res.Error, io.EOF) {
		return Response{
			Variant: ResponseVariantAck,
			Error:   res.Error,
			Command: EOF,
			Ack:     res.Error == io.EOF,
		}, true, nil
	}
	return Response{
		Variant:  ResponseVariantData,
		Error:    res.Error,
		Segments: te.wrapper.Wrap(res.Segments),
	}, true, nil
}
