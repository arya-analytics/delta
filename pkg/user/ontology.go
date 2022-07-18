package user

import (
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/delta/pkg/ontology/schema"
	"github.com/google/uuid"
)

const ontologyType ontology.Type = "user"

func OntologyID(key uuid.UUID) ontology.ID {
	return ontology.ID{Type: ontologyType, Key: key.String()}
}

var userSchema = &ontology.Schema{
	Type: ontologyType,
	Fields: map[string]schema.Field{
		"key":      {Type: schema.String},
		"username": {Type: schema.String},
	},
}

var _ ontology.Service = (*Service)(nil)

// Schema implements the ontology.Service interface.
func (s *Service) Schema() *schema.Schema { return userSchema }

// RetrieveEntity implements the ontology.Service interface.
func (s *Service) RetrieveEntity(key string) (schema.Entity, error) {
	uuidKey, err := uuid.Parse(key)
	if err != nil {
		return schema.Entity{}, err
	}
	u, err := s.Retrieve(uuidKey)
	return newEntity(u), err
}

func newEntity(u User) schema.Entity {
	e := schema.NewEntity(userSchema)
	schema.Set(e, "key", u.Key.String())
	schema.Set(e, "username", u.Username)
	return e
}
