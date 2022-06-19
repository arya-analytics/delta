package channel

import "github.com/arya-analytics/aspen"

func BatchByNodeID(channels []Channel) map[aspen.NodeID][]Channel {
	m := make(map[aspen.NodeID][]Channel)
	for _, ch := range channels {
		m[ch.NodeID] = append(m[ch.NodeID], ch)
	}
	return m
}
