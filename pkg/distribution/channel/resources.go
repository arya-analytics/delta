package channel

import (
	"context"
	"github.com/arya-analytics/delta/pkg/ontology"
)

const ResourceType ontology.Type = "channel"

type ResourceProvider struct {
	svc *Service
}

func (rp *ResourceProvider) GetAttributes(key string) (ontology.Attributes, error) {
	k, err := ParseKey(key)
	if err != nil {
		return ontology.Attributes{}, err
	}
	var ch Channel
	err = rp.svc.NewRetrieve().
		WhereKeys(k).
		Entry(&ch).
		Exec(context.TODO())
	return ontology.Attributes{
		Name: ch.Name,
		Extra: map[string]interface{}{
			"nodeID":   ch.NodeID,
			"dataRate": ch.Cesium.DataRate,
			"dataType": ch.Cesium.DataType,
		},
	}, err
}
