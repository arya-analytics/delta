package ontology

import (
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/query"
)

type Retrieve struct {
	txn   gorp.Txn
	query *gorp.Compound[ID, Resource]
}

func newRetrieve(txn gorp.Txn) Retrieve {
	r := Retrieve{
		txn:   txn,
		query: &gorp.Compound[ID, Resource]{},
	}
	r.query.Next()
	return r
}

// WhereIDs filters resources by the provided keys.
func (r Retrieve) WhereIDs(keys ...ID) Retrieve {
	r.query.Current().WhereKeys(keys...)
	return r
}

func (r Retrieve) Where(filter func(r *Resource) bool) Retrieve {
	r.query.Current().Where(filter)
	return r
}

type Direction uint8

const (
	Forward  Direction = 1
	Backward Direction = 2
)

type Traverser struct {
	Filter    func(res *Resource, rel *Relationship) bool
	Direction Direction
}

var (
	Children = Traverser{
		Filter: func(res *Resource, rel *Relationship) bool {
			return rel.Type == Parent && rel.To == res.ID
		},
		Direction: Backward,
	}
	Parents = Traverser{
		Filter: func(res *Resource, rel *Relationship) bool {
			return rel.Type == Parent && rel.From == res.ID
		},
		Direction: Forward,
	}
)

// TraverseTo traverses to the provided relationship type. All filtering methods will
// now be applied to elements of the traversed relationship.
func (r Retrieve) TraverseTo(t Traverser) Retrieve {
	setTraverser(r.query.Current(), t)
	r.query.Next()
	return r
}

// Entry binds the entry that the Query will fill results into. Calls to Entry will
// override all previous calls to Entries or Entry.
func (r Retrieve) Entry(res *Resource) Retrieve {
	r.query.Current().Entry(res)
	return r
}

// Entries binds a slice that the Query will fill results into. Calls to Entry will
// override all previous calls to Entries or Entry.
func (r Retrieve) Entries(res *[]Resource) Retrieve {
	r.query.Current().Entries(res)
	return r
}

func (r Retrieve) Exec() error { return retrieve{txn: r.txn}.exec(r) }

const traverseOptKey = "traverse"

func setTraverser(q query.Query, f Traverser) { q.Set(traverseOptKey, f) }

func getTraverser(q query.Query) Traverser { return q.GetRequired(traverseOptKey).(Traverser) }

type retrieve struct{ txn gorp.Txn }

func (r retrieve) exec(q Retrieve) error {
	var nextIDs []ID
	for i, step := range q.query.Clauses {
		if i != 0 {
			step.WhereKeys(nextIDs...)
		}
		nextIDs = nil
		if err := step.Exec(r.txn); err != nil {
			return err
		}
		entries := gorp.GetEntries[ID, Resource](step).All()
		if len(entries) == 0 {
			break
		}
		if len(q.query.Clauses)-1 == i {
			return nil
		}
		traverse := getTraverser(step)
		if err := gorp.NewRetrieve[string, Relationship]().Where(func(rel *Relationship) bool {
			for _, entry := range entries {
				if traverse.Filter(&entry, rel) {
					if traverse.Direction == Forward {
						nextIDs = append(nextIDs, rel.To)
					} else {
						nextIDs = append(nextIDs, rel.From)
					}
					break
				}
			}
			return false
		}).Entries(&[]Relationship{}).Exec(r.txn); err != nil {
			return err
		}
	}
	return nil
}
