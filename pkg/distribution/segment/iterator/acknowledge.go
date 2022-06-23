package iterator

import (
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/filter"
)

type acknowledge struct {
	nodeIDs []node.ID
	confluence.UnarySink[Response]
}

func (a *acknowledge) acknowledge() bool {
	var acknowledgements []node.ID
	for r := range a.In.Outlet() {
		if !filter.ElementOf(acknowledgements, r.NodeID) {
			// If any node does not acknowledge the request as valid, then we consider
			// the entire command as invalid.
			if !r.Ack {
				return false
			}
			acknowledgements = append(acknowledgements, r.NodeID)
		}
		if len(acknowledgements) == len(a.nodeIDs) {
			return true
		}
	}
	panic(
		"[iterator.acknowledge] - response pipe closed before all nodes acked command",
	)
}
