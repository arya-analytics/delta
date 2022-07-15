package user

import (
	"github.com/arya-analytics/delta/pkg/resource"
	"github.com/google/uuid"
)

const ResourceType = "user"

func ResourceKey(key uuid.UUID) resource.Key {
	return resource.Key{Type: ResourceType, Key: key.String()}
}

type ResourceProvider struct {
	svc *Service
}

func (rp *ResourceProvider) GetAttributes(key string) (resource.Attributes, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return resource.Attributes{}, err
	}
	user, err := rp.svc.Retrieve(k)
	if err != nil {
		return resource.Attributes{}, err
	}
	return resource.Attributes{Name: user.Username}, err
}
