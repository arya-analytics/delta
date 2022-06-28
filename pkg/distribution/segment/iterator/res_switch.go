package iterator

import (
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
)

type responseFilter struct {
	confluence.Filter[Response]
}

func newResponseFilter(rejects confluence.Inlet[Response]) responseSegment {
	rs := &responseFilter{}
	rs.Filter.Rejects = rejects
	rs.Filter.Filter = rs.filter
	return rs
}

func (rs *responseFilter) filter(ctx signal.Context, res Response) (bool, error) {
	return res.Variant == ResponseVariantData, nil
}
