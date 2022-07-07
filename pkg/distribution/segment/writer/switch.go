package writer

import (
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/delta/pkg/proxy"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/transfluence"
	"github.com/arya-analytics/x/signal"
)

type requestSwitchSender struct {
	transfluence.BatchSwitchSender[Request, Request]
	addresses proxy.AddressMap
}

func newRequestSwitchSender(
	addresses proxy.AddressMap,
) *requestSwitchSender {
	rs := &requestSwitchSender{addresses: addresses}
	rs.BatchSwitchSender.ApplySwitch = rs._switch
	return rs
}

func (rs *requestSwitchSender) _switch(ctx signal.Context,
	r Request, oReqs map[address.Address]Request) error {
	for _, seg := range r.Segments {
		addr := rs.addresses[seg.ChannelKey.NodeID()]
		oReqs[addr] = Request{Segments: append(oReqs[addr].Segments, seg)}
	}
	return nil
}

type remoteLocalSwitch struct {
	confluence.BatchSwitch[Request, Request]
	host node.ID
}

func newRemoteLocalSwitch(
	host node.ID,
) *remoteLocalSwitch {
	rl := &remoteLocalSwitch{host: host}
	rl.ApplySwitch = rl._switch
	return rl
}

func (rl *remoteLocalSwitch) _switch(
	ctx signal.Context,
	r Request,
	oReqs map[address.Address]Request,
) error {
	for _, seg := range r.Segments {
		if seg.ChannelKey.NodeID() == rl.host {
			oReqs["local"] = Request{Segments: append(oReqs["local"].Segments, seg)}
		} else {
			oReqs["remote"] = Request{Segments: append(oReqs["remote"].Segments, seg)}
		}
	}
	return nil
}
