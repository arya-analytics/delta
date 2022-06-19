package channel

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/x/address"
)

// Resolver is a type that can resolve the address of a node from the key of a channel.
type Resolver interface {
	// Resolve resolves the address for a node from the key of a channel.
	Resolve(key Key) (address.Address, error)
}

type resolver struct {
	// Cluster is the cluster to resolve addresses for.
	cluster aspen.Cluster
}

// Resolve resolves an address for a node from the key of a channel.
func (r *resolver) Resolve(key Key) (address.Address, error) {
	return r.cluster.Resolve(key.NodeID())
}
