package iterator

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence/transfluence"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/telem"
)

func openRemoteIterators(
	ctx signal.Context,
	tran Transport,
	targets map[node.ID][]channel.Key,
	rng telem.TimeRange,
	resolver aspen.HostResolver,
) (*transfluence.MultiSender[Request], []*transfluence.Receiver[Response], error) {
	sender := &transfluence.MultiSender[Request]{}
	receivers := make([]*transfluence.Receiver[Response], 0, len(targets))
	for nodeID, keys := range targets {
		targetAddr, err := resolver.Resolve(nodeID)
		if err != nil {
			return sender, receivers, err
		}
		client, err := openRemoteClient(ctx, tran, targetAddr, keys, rng)
		if err != nil {
			return sender, receivers, err
		}
		sender.Senders = append(sender.Senders, client)
		receivers = append(receivers, &transfluence.Receiver[Response]{Receiver: client})
	}
	return sender, receivers, nil
}

func openRemoteClient(
	ctx signal.Context,
	tran Transport,
	target address.Address,
	keys channel.Keys,
	rng telem.TimeRange,
) (Client, error) {
	client, err := tran.Stream(ctx, target)
	if err != nil {
		return nil, err
	}

	// Send an open request to the transport. This will open a localIterator  on the
	// target node.
	return client, client.Send(Request{
		Command: Open,
		Keys:    keys,
		Range:   rng,
	})
}
