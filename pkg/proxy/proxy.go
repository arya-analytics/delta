package proxy

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/x/address"
)

type Entry interface {
	Lease() aspen.NodeID
}

type BatchFactory[E Entry] interface {
	Batch(entries []E) Batch[E]
}

type batchFactory[E Entry] struct {
	host aspen.NodeID
}

type Batch[E Entry] struct {
	Local  []E
	Remote map[node.ID][]E
}

func NewBatchFactory[E Entry](host aspen.NodeID) BatchFactory[E] { return batchFactory[E]{host} }

func (p batchFactory[E]) Batch(entries []E) Batch[E] {
	b := Batch[E]{Remote: make(map[node.ID][]E)}
	for _, entry := range entries {
		lease := entry.Lease()
		if lease == p.host {
			b.Local = append(b.Local, entry)
		} else {
			b.Remote[lease] = append(b.Remote[lease], entry)
		}
	}
	return b
}

type AddressMap map[node.ID]address.Address

func ResolveAddressMap(resolver aspen.Resolver, nodes ...node.ID) (AddressMap, error) {
	addrMap := make(AddressMap, len(nodes))
	for _, id := range nodes {
		addr, err := resolver.Resolve(id)
		if err != nil {
			return addrMap, err
		}
		addrMap[id] = addr
	}
	return addrMap, nil
}
