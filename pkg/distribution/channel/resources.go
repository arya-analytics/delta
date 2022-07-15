package channel

import (
	"context"
	"github.com/arya-analytics/delta/pkg/resource"
)

const ResourceType resource.Type = "channel"

type ResourceProvider struct {
	svc *Service
}

func (rp *ResourceProvider) GetAttributes(key string) (resource.Attributes, error) {
	k, err := ParseKey(key)
	if err != nil {
		return resource.Attributes{}, err
	}
	var ch Channel
	err = rp.svc.NewRetrieve().
		WhereKeys(k).
		Entry(&ch).
		Exec(context.TODO())
	return resource.Attributes{
		Name: ch.Name,
		Extra: map[string]interface{}{
			"nodeID":   ch.NodeID,
			"dataRate": ch.Cesium.DataRate,
			"dataType": ch.Cesium.DataType,
		},
	}, err
}
