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
	db       *gorp.DB
	retrieve retrieve
}

// Open opens the ontology stored in the given database.
func Open(db *gorp.DB) (*Ontology, error) {
	o := &Ontology{
		db:       db,
		retrieve: retrieve{services: make(services)},
	}
	if err := o.NewWriter(db).DefineResource(Root); err != nil {
		return nil, err
	}
	return o, nil
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
func (o *Ontology) NewRetrieve() Retrieve { return newRetrieve(o.db, o.retrieve.exec) }

// NewWriter opens a new Writer using the provided transaction. NewWriter will panic
// if the transaction does not root from the same database as the Ontology.
func (o *Ontology) NewWriter(txn gorp.Txn) Writer { return dagWriter{txn: txn, retrieve: o.retrieve} }

func (o *Ontology) RegisterService(s Service) {
	o.retrieve.services.Register(s)
}
