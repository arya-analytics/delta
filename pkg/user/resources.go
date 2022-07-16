package user

import (
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/google/uuid"
)

const ResourceType = "user"

func ResourceKey(key uuid.UUID) ontology.Key {
	return ontology.Key{Type: ResourceType, Key: key.String()}
}

type ResourceProvider struct {
	svc *Service
}

func (rp *ResourceProvider) GetAttributes(key string) (ontology.Attributes, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Attributes{}, err
	}
	user, err := rp.svc.Retrieve(k)
	if err != nil {
		return ontology.Attributes{}, err
	}
	return ontology.Attributes{Name: user.Username}, err
}
