package writer

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/delta/pkg/proxy"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence/transfluence"
	"github.com/arya-analytics/x/signal"
)

func openRemoteWriters(
	ctx signal.Context,
	tran Transport,
	targets map[node.ID][]channel.Key,
	resolver aspen.HostResolver,
) (*requestSwitchSender,
	[]*transfluence.Receiver[Response], error) {
	receivers := make([]*transfluence.Receiver[Response], 0, len(targets))
	addrMap := make(proxy.AddressMap)
	sender := newRequestSwitchSender(addrMap)
	for nodeID, keys := range targets {
		targetAddr, err := resolver.Resolve(nodeID)
		if err != nil {
			return sender, receivers, err
		}
		addrMap[nodeID] = targetAddr
		client, err := openRemoteClient(ctx, tran, targetAddr, keys)
		if err != nil {
			return sender, receivers, err
		}
		sender.Senders[targetAddr] = client
		receivers = append(receivers, &transfluence.Receiver[Response]{Receiver: client})
	}
	return sender, receivers, nil
}

func openRemoteClient(
	ctx signal.Context,
	tran Transport,
	target address.Address,
	keys channel.Keys,
) (Client, error) {
	client, err := tran.Stream(ctx, target)
	if err != nil {
		return nil, err
	}
	return client, client.Send(Request{OpenKeys: keys})
}
