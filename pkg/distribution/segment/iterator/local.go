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
)

type localIterator struct {
	req       confluence.Sink[Request]
	res       confluence.Source[Response]
	cesiumRes confluence.Segment[cesium.RetrieveResponse]
}

func (s *localIterator) Flow(ctx signal.Context, opts ...confluence.FlowOption) {
	s.req.Flow(ctx)
	s.res.Flow(ctx)
	s.cesiumRes.Flow(ctx)
}

func (s *localIterator) OutTo(inlets ...confluence.Inlet[Response]) { s.res.OutTo(inlets...) }

func (s *localIterator) InFrom(outlets ...confluence.Outlet[Request]) { s.req.InFrom(outlets...) }

func newLocalIterator(
	db cesium.DB,
	host node.ID,
	rng telem.TimeRange,
	keys channel.Keys,
) (confluence.Translator[Request, Response], error) {
	iter := db.NewRetrieve().WhereTimeRange(rng).WhereChannels(keys.Cesium()...).Iterate()
	if iter.Error() != nil {
		return nil, errors.Wrap(iter.Error(), "[segment.iterator] - server failed to open cesium iterator")
	}

	// to a point where they can be translated into a transport Response.
	cesiumRes := confluence.NewPipeline[cesium.RetrieveResponse]()
	req := confluence.NewPipeline[Request]()
	res := confluence.NewPipeline[Response]()

	// cesiumRes receives segments from the iterator.
	cesiumRes.Source("iterator", iter)

	// executor executes req as method calls on the iterator. Pipes
	// synchronous acknowledgements out to the response pipeline.
	te := newRequestExecutor(host, iter)
	req.Sink("executor", te)
	res.Source("executor", te)

	// translator translates cesium res from the iterator source into
	// res transportable over the network.
	ts := newCesiumResponseTranslator(keys.CesiumMap())
	cesiumRes.Sink("translator", ts)
	res.Source("translator", ts)

	reqBuilder := req.NewRouteBuilder()
	resBuilder := res.NewRouteBuilder()
	cesiumBuilder := cesiumRes.NewRouteBuilder()

	cesiumBuilder.RouteUnary("iterator", "translator", 0)
	reqBuilder.RouteInletTo("executor")
	resBuilder.RouteOutletFrom("executor", "translator")

	reqBuilder.PanicIfErr()
	resBuilder.PanicIfErr()
	cesiumBuilder.PanicIfErr()

	return &localIterator{req: req, res: res, cesiumRes: cesiumRes}, nil
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

func newCesiumResponseTranslator(
	keyMap map[cesium.ChannelKey]channel.Key,
) confluence.Translator[cesium.RetrieveResponse, Response] {
	wrapper := &core.CesiumWrapper{KeyMap: keyMap}
	ts := &cesiumResponseTranslator{wrapper: wrapper}
	ts.CoreTranslator.Translate = ts.translate
	return confluence.GateTranslator[cesium.RetrieveResponse, Response](ts)
}

func (te *cesiumResponseTranslator) translate(
	ctx signal.Context,
	res cesium.RetrieveResponse,
) (Response, bool, error) {
	return Response{Variant: DataResponse, Segments: te.wrapper.Wrap(res.Segments)}, true, nil
}
