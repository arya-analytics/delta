package proxy

import "github.com/arya-analytics/aspen"

type Entry interface {
	Lease() aspen.NodeID
}

type BatchFactory[E Entry] interface {
	Batch(entries []E) (local []E, remote map[aspen.NodeID][]E)
}

type batch[E Entry] struct {
	host aspen.NodeID
}

func NewBatchFactory[E Entry](host aspen.NodeID) BatchFactory[E] { return batch[E]{host} }

func (p batch[E]) Batch(entries []E) (local []E, remote map[aspen.NodeID][]E) {
	local = make([]E, 0, len(entries))
	remote = make(map[aspen.NodeID][]E)
	for _, entry := range entries {
		lease := entry.Lease()
		if lease == p.host {
			local = append(local, entry)
		} else {
			remote[lease] = append(remote[lease], entry)
		}
	}
	return local, remote
}
