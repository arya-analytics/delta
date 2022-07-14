package channel

import (
	"context"
	"github.com/arya-analytics/delta/pkg/resource"
	"github.com/arya-analytics/x/gorp"
)

const (
	ResourceType resource.Type = "channel"
)

type ResourceProvider struct {
	svc *Service
}

func (rp *ResourceProvider) GetAttributes(txn gorp.Txn, key string) (resource.Attributes, error) {
	k, err := ParseKey(key)
	if err != nil {
		return resource.Attributes{}, err
	}
	var ch Channel
	if err := rp.svc.NewRetrieve().
		WhereKeys(k).
		Entry(&ch).
		WithTxn(txn).
		Exec(context.TODO()); err != nil {
		return resource.Attributes{}, err
	}
	return resource.Attributes{
		Name: ch.Name,
		Extra: map[string]interface{}{
			"nodeID":   ch.NodeID,
			"dataRate": ch.Cesium.DataRate,
			"dataType": ch.Cesium.DataType,
		},
	}, nil
}
