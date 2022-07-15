package resource

import (
	"github.com/arya-analytics/x/gorp"
)

type Service struct {
	Providers providers
}

func OpenService(txn gorp.Txn) (*Service, error) {
	s := &Service{
		Providers: map[Type]Provider{},
	}
	if err := s.NewWriter(txn).DefineResource(RootKey); err != nil {
		return nil, err
	}
	return s, nil
}

const BuiltIn Type = "builtin"

var RootKey = Key{Type: BuiltIn, Key: "root"}

type Writer interface {
	DefineResource(key Key) error
	DeleteResource(key Key) error
	DefineRelationship(parent, child Key) error
	DeleteRelationship(parent, child Key) error
}

type Reader interface {
	GetResource(key Key) (Resource, error)
	GetChildResources(key Key) ([]Resource, error)
	GetParentResources(key Key) ([]Resource, error)
}

func (s *Service) NewReader(txn gorp.Txn) Reader {
	return attributeReader{Providers: s.Providers, dag: DAG{Txn: txn}}
}

func (s *Service) NewWriter(txn gorp.Txn) Writer {
	return DAG{Txn: txn}
}

func (s *Service) RegisterProvider(t Type, p Provider) { s.Providers[t] = p }

type providers map[Type]Provider

func (p providers) Get(t Type) Provider {
	prov, ok := p[t]
	if !ok {
		panic("[resource] - provider not found")
	}
	return prov
}

func (p providers) GetAttributes(txn gorp.Txn, key Key) (Attributes, error) {
	return p.Get(key.Type).GetAttributes(txn, key.Key)
}
