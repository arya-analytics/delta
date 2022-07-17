package user

import (
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/google/uuid"
)

const ResourceType = "user"

func ResourceKey(key uuid.UUID) ontology.ID {
	return ontology.ID{Type: ResourceType, Key: key.String()}
}

type ResourceProvider struct {
	svc *Service
}

func (rp *ResourceProvider) Retrieve(key string) (ontology.Data, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Data{}, err
	}
	user, err := rp.svc.Retrieve(k)
	if err != nil {
		return ontology.Data{}, err
	}
	return ontology.Data{Name: user.Username}, err
}
