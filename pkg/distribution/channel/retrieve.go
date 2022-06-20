package channel

import (
	"context"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/x/gorp"
)

type Retrieve struct {
	gorp gorp.Retrieve[Key, Channel]
	db   *gorp.DB
}

func newRetrieve(db *gorp.DB) Retrieve {
	return Retrieve{
		gorp: gorp.NewRetrieve[Key, Channel](),
		db:   db,
	}
}

func (r Retrieve) Entry(ch *Channel) Retrieve { r.gorp.Entry(ch); return r }

func (r Retrieve) Entries(ch *[]Channel) Retrieve { r.gorp.Entries(ch); return r }

func (r Retrieve) WhereNodeID(nodeID aspen.NodeID) Retrieve {
	r.gorp.Where(func(ch Channel) bool {
		return ch.NodeID == nodeID
	})
	return r
}

func (r Retrieve) WhereKeys(keys ...Key) Retrieve {
	r.gorp.WhereKeys(keys...)
	return r
}

func (r Retrieve) Exec(ctx context.Context) error { return r.gorp.Exec(r.db) }
