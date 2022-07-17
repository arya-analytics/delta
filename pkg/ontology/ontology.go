package ontology

import (
	"github.com/arya-analytics/x/gorp"
)

type Ontology struct {
	Services services
	DB       *gorp.DB
}

func Open(db *gorp.DB) (*Ontology, error) {
	s := &Ontology{Services: make(services)}
	if err := s.NewWriter(db).DefineResource(RootKey); err != nil {
		return nil, err
	}
	return s, nil
}

type Writer interface {
	DefineResource(key Key) error
	DeleteResource(key Key) error
	DefineRelationship(parent, child Key) error
	DeleteRelationship(parent, child Key) error
}

type Reader interface {
	RetrieveResource(key Key) (Resource, error)
	RetrieveChildResources(key Key) ([]Resource, error)
	RetrieveParentResources(key Key) ([]Resource, error)
}

func (s *Ontology) NewReader() Reader {
	return attributeReader{Providers: s.Services, dag: DAG{DB: s.DB}}
}

func (s *Ontology) NewWriter(txn gorp.Txn) Writer {
	return DAG{DB: txn}
}
