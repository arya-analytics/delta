package ontology

import (
	"github.com/arya-analytics/delta/pkg/ontology/schema"
	"github.com/arya-analytics/x/gorp"
)

type (
	Schema = schema.Schema
	Entity = schema.Entity
	Type   = schema.Type
)

type Ontology struct {
	services services
	db       *gorp.DB
}

// Open opens the ontology stored in the given database.
func Open(db *gorp.DB) (*Ontology, error) {
	s := &Ontology{services: make(services), db: db}
	if err := s.NewWriter(db).DefineResource(Root); err != nil {
		return nil, err
	}
	return s, nil
}

type Writer interface {
	// DefineResource defines a new resource with the given ID. If the resource already
	// exists, DefineResource does nothing.
	DefineResource(id ID) error
	// DeleteResource deletes the resource with the given ID along with all of its
	// incoming and outgoing relationships.  If the resource does not exist,
	// DeleteResource does nothing.
	DeleteResource(id ID) error
	// DefineRelationship defines a directional relationship of type t between the
	// resources with the given IDs. If the relationship already exists, DefineRelationship
	// does nothing.
	DefineRelationship(from, to ID, t RelationshipType) error
	// DeleteRelationship deletes the relationship with the given IDs and type. If the
	// relationship does not exist, DeleteRelationship does nothing.
	DeleteRelationship(from, to ID, t RelationshipType) error
	// NewRetrieve opens a new Retrieve query that uses the Writers transaction.
	NewRetrieve() Retrieve
}

// NewRetrieve opens a new Retrieve query, which is used to traverse the ontology.
func (s *Ontology) NewRetrieve() Retrieve { return newRetrieve(s.db) }

// NewWriter opens a new Writer using the provided transaction. NewWriter will panic
// if the transaction does not root from the same database as the Ontology.
func (s *Ontology) NewWriter(txn gorp.Txn) Writer { return dagWriter{Txn: txn} }
