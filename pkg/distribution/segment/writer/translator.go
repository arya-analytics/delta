package writer

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
)

type requestTranslator struct {
	confluence.LinearTransform[Request, cesium.CreateRequest]
}

func newRequestTranslator() *requestTranslator {
	rt := &requestTranslator{}
	rt.ApplyTransform = rt.translate
	return rt
}

func (rt *requestTranslator) translate(
	ctx signal.Context,
	in Request,
) (cesium.CreateRequest, bool, error) {
	req := cesium.CreateRequest{Segments: make([]cesium.Segment, len(in.Segments))}
	for i, seg := range in.Segments {
		req.Segments[i] = seg.Segment
	}
	return req, true, nil
}

type responseTranslator struct {
	confluence.LinearTransform[cesium.CreateResponse, Response]
}

func (rt *responseTranslator) translate(
	ctx signal.Context,
	in cesium.CreateResponse,
) (Response, bool, error) {
	return Response(in), true, nil
}

func newResponseTranslator() *responseTranslator {
	rt := &responseTranslator{}
	rt.ApplyTransform = rt.translate
	return rt
}
