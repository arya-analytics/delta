package writer

import (
	"github.com/arya-analytics/delta/pkg/proxy"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence/transfluence"
	"github.com/arya-analytics/x/signal"
)

type requestSwitchSender struct {
	transfluence.BatchSwitchSender[Request, Request]
	addresses proxy.AddressMap
}

func (rs *requestSwitchSender) _switch(ctx signal.Context,
	r Request, oReqs map[address.Address]Request) error {
	for _, seg := range r.Segments {
		addr := rs.addresses[seg.ChannelKey.NodeID()]
		oReqs[addr] = Request{Segments: append(oReqs[addr].Segments, seg)}
	}
	return nil
}

func newRequestSwitchSender(
	addresses proxy.AddressMap,
) *requestSwitchSender {
	rs := &requestSwitchSender{addresses: addresses}
	rs.BatchSwitchSender.ApplySwitch = rs._switch
	return rs
}
