package iterator

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/filter"
	"github.com/arya-analytics/x/signal"
	"github.com/sirupsen/logrus"
	"time"
)

type synchronizer struct {
	timeout time.Duration
	nodeIDs []node.ID
	confluence.UnarySink[Response]
}

func (a *synchronizer) sync(ctx context.Context, command Command) bool {
	ctx, cancel := signal.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	acknowledgements := make([]node.ID, 0, len(a.nodeIDs))
	logrus.Infof("Listening for acknowledgements for command %v ", command)
	for {
		select {
		case <-ctx.Done():
			return false
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
				logrus.Infof("Receive ack from %s for command %v", r.NodeID, command)
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
