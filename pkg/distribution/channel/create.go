package channel

import (
	"context"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/x/query"
	"github.com/arya-analytics/x/telem"
)

type Create struct {
	query.Query
	proxy *leaseProxy
}

func newCreate(proxy *leaseProxy) Create {
	return Create{Query: query.New(), proxy: proxy}
}

func (c Create) WithNodeID(nodeID aspen.NodeID) Create { setNodeID(c, nodeID); return c }

func (c Create) WithName(name string) Create { setName(c, name); return c }

func (c Create) WithDataRate(dr telem.DataRate) Create { telem.SetDataRate(c, dr); return c }

func (c Create) WithDataType(dt telem.DataType) Create { telem.SetDataType(c, dt); return c }

func (c Create) Exec(ctx context.Context) (Channel, error) {
	channels, err := c.ExecN(ctx, 1)
	if err != nil {
		return Channel{}, err
	}
	return channels[0], nil
}

func (c Create) ExecN(ctx context.Context, n int) ([]Channel, error) {
	channels, err := assembleFromQuery(c, n)
	if err != nil {
		return channels, err
	}
	return c.proxy.create(ctx, channels)
}

func assembleFromQuery(q query.Query, n int) ([]Channel, error) {
	channels := make([]Channel, n)
	dr, err := telem.GetDataRate(q)
	if err != nil {
		return channels, err
	}
	dt, err := telem.GetDataType(q)
	if err != nil {
		return channels, err
	}
	name := getName(q)
	nodeID := getNodeID(q)
	for i := 0; i < n; i++ {
		channels[i] = Channel{
			Name:   name,
			NodeID: nodeID,
			Cesium: cesium.Channel{DataRate: dr, DataType: dt},
		}
	}
	return channels, nil
}

// |||||| LEASE ||||||

const nodeIDKey query.OptionKey = "nodeID"

func setNodeID(q query.Query, nodeID aspen.NodeID) { q.Set(nodeIDKey, nodeID) }

func getNodeID(q query.Query) aspen.NodeID {
	if v, ok := q.Get(nodeIDKey); ok {
		return v.(aspen.NodeID)
	}
	return 0
}

// |||||| NAME ||||||

const nameKey query.OptionKey = "name"

func setName(q query.Query, name string) { q.Set(nameKey, name) }

func getName(q query.Query) string {
	if v, ok := q.Get(nameKey); ok {
		return v.(string)
	}
	return ""
}
