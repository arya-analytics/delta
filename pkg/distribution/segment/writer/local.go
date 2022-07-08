package writer

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
)

func newLocalWriter(
	ctx context.Context,
	db cesium.DB,
	keys channel.Keys,
) (confluence.Segment[Request, Response], error) {
	requests, responses, err := db.NewCreate().WhereChannels(keys.Cesium()...).Stream(ctx)
	if err != nil {
		return nil, err
	}
	reqT := newRequestTranslator()
	reqT.OutTo(confluence.NewInlet[cesium.CreateRequest](requests))
	resT := newResponseTranslator()
	resT.InFrom(confluence.NewOutlet[cesium.CreateResponse](responses))
	pipe := plumber.New()
	plumber.SetSegment[Request, cesium.CreateRequest](pipe, "requestTranslator", reqT)
	plumber.SetSegment[cesium.CreateResponse, Response](pipe, "responseTranslator", resT)
	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	if err := seg.RouteInletTo("requestTranslator"); err != nil {
		panic(err)
	}
	if err := seg.RouteOutletFrom("responseTranslator"); err != nil {
		panic(err)
	}
	return seg, nil
}
