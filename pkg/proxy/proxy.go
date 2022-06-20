package proxy

import "github.com/arya-analytics/aspen"

type Entry interface {
	Lease() aspen.NodeID
}

type Router[E Entry] interface {
	Route(entries []E) (local []E, remote map[aspen.NodeID][]E)
}

type proxy[E Entry] struct {
	host aspen.NodeID
}

func NewRouter[E Entry](host aspen.NodeID) Router[E] { return proxy[E]{host} }

func (p proxy[E]) Route(entries []E) (local []E, remote map[aspen.NodeID][]E) {
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
