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

func (a *synchronizer) sync(
	ctx context.Context,
	command Command) bool {
	_, ok := a.syncWithRes(ctx, command)
	return ok
}

func (a *synchronizer) syncWithRes(
	ctx context.Context,
	command Command,
) ([]Response, bool) {
	ctx, cancel := signal.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	acknowledgements := make([]node.ID, 0, len(a.nodeIDs))
	responses := make([]Response, 0, len(a.nodeIDs))
	for {
		select {
		case <-ctx.Done():
			return responses, false
		case r, ok := <-a.In.Outlet():
			if r.Command != command {
				continue
			}
			if !ok {
				panic(
					"[iterator.synchronizer] - response pipe closed before all nodes acked command",
				)
			}
			if !filter.ElementOf(acknowledgements, r.NodeID) {
				// If any node does not synchronizer the request as valid, then we consider
				// the entire command as invalid.
				if !r.Ack {
					return responses, false
				}
				acknowledgements = append(acknowledgements, r.NodeID)
				responses = append(responses, r)
			}
			if len(acknowledgements) == len(a.nodeIDs) {
				return responses, true
			}
		}
	}
}
