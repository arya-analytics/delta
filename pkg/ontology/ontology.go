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
	Services services
	DB       *gorp.DB
}

func Open(db *gorp.DB) (*Ontology, error) {
	s := &Ontology{Services: make(services), DB: db}
	if err := s.NewWriter(db).DefineResource(Root); err != nil {
		return nil, err
	}
	return s, nil
}

type Writer interface {
	DefineResource(key ID) error
	DeleteResource(key ID) error
	DefineRelationship(from, to ID, t RelationshipType) error
	DeleteRelationship(from, to ID, t RelationshipType) error
}

func (s *Ontology) NewRetrieve() Retrieve {
	return newRetrieve(retrieve{db: s.DB}.exec)
}

func (s *Ontology) NewWriter(txn gorp.Txn) Writer {
	return DAG{DB: txn}
}
