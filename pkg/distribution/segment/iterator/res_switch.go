package iterator

import (
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
)

type responseSwitch struct {
	confluence.Switch[Response]
}

func newResponseSwitch() responseSegment {
	rs := &responseSwitch{}
	rs.Switch.Switch = rs._switch
	return rs
}

func (rs *responseSwitch) _switch(ctx confluence.Context, res Response) address.Address {
	switch res.Variant {
	case ResponseVariantData:
		return acknowledgeAddr
	case ResponseVariantAck:
		return dataAddr
	default:
		return dataAddr
	}
}
