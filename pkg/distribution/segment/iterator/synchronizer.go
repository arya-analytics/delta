package iterator

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/filter"
	"github.com/arya-analytics/x/signal"
	"time"
)

type synchronizer struct {
	timeout time.Duration
	nodeIDs []node.ID
	confluence.UnarySink[Response]
}

func (a *synchronizer) acknowledge(ctx context.Context) bool {
	ctx, cancel := signal.WithTimeout(ctx, a.timeout)
	defer cancel()
	var acknowledgements []node.ID
	for {
		select {
		case <-ctx.Done():
			return false
		case r, ok := <-a.In.Outlet():
			if !ok {
				panic(
					"[iterator.synchronizer] - response pipe closed before all nodes acked command",
				)
			}
			if !filter.ElementOf(acknowledgements, r.NodeID) {
				// If any node does not synchronizer the request as valid, then we consider
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
	}
}
