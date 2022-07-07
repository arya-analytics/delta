package writer

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/signal"
)

func newLocalWriter(
	db cesium.DB,
	keys channel.Keys,
) confluence.Segment[Request, Response] {
	cw := newCesiumWriter(db, keys)
	reqT := newRequestTranslator()
	resT := newResponseTranslator()

	pipe := plumber.New()
	plumber.SetSegment[Request, cesium.CreateRequest](pipe, "requestTranslator", reqT)
	plumber.SetSegment[cesium.CreateResponse, Response](pipe, "responseTranslator", resT)
	plumber.SetSegment[cesium.CreateRequest, cesium.CreateResponse](pipe, "cesiumWriter", cw)

	c := errutil.NewCatchSimple()

	c.Exec(plumber.UnaryRouter[cesium.CreateRequest]{
		SourceTarget: "requestTranslator",
		SinkTarget:   "cesiumWriter",
	}.PreRoute(pipe))

	c.Exec(plumber.UnaryRouter[cesium.CreateResponse]{
		SourceTarget: "cesiumWriter",
		SinkTarget:   "responseTranslator",
	}.PreRoute(pipe))

	if c.Error() != nil {
		panic(c.Error())
	}

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}

	if err := seg.RouteInletTo("requestTranslator"); err != nil {
		panic(err)
	}

	if err := seg.RouteOutletFrom("responseTranslator"); err != nil {
		panic(err)
	}

	return seg
}

func newCesiumWriter(
	db cesium.DB,
	keys channel.Keys,
) confluence.Segment[cesium.CreateRequest, cesium.CreateResponse] {
	return &cesiumWriter{query: db.NewCreate().WhereChannels(keys.Cesium()...)}
}

type cesiumWriter struct {
	confluence.LinearTransform[cesium.CreateRequest, cesium.CreateResponse]
	query cesium.Create
}

func (cw *cesiumWriter) Flow(ctx signal.Context, opts ...confluence.Option) {
	ctx.Go(func() error {
		return cw.query.Stream(ctx, cw.In.Outlet(), cw.Out.Inlet())
	}, confluence.NewOptions(opts).Signal...)
}
