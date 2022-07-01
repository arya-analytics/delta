package iterator

import (
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
)

type ackFilter struct {
	confluence.Filter[Response]
}

func newAckRouter(ackMessages confluence.Inlet[Response]) *ackFilter {
	rs := &ackFilter{}
	rs.Filter.Rejects = ackMessages
	rs.Filter.Filter = rs.filter
	return rs
}

func (rs *ackFilter) filter(ctx signal.Context, res Response) (bool, error) {
	return res.Variant == DataResponse, nil
}
