package channel

import (
	"context"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/query"
	"github.com/arya-analytics/x/telem"
)

// Create is used to create a new Channel in delta's distribution layer.
type Create struct {
	query.Query
	proxy *leaseProxy
}

func newCreate(proxy *leaseProxy) Create {
	return Create{Query: query.New(), proxy: proxy}
}

// WithNodeID lets the leaseholder node for the Channel. If this option is not set,
// the leaseholder is assumed to be the host. If the provided node is not the host
// Exec and ExecN will execute as a remote RPC on the leaseholder to guarantee
// consistency.
func (c Create) WithNodeID(nodeID aspen.NodeID) Create { setNodeID(c, nodeID); return c }

// WithName sets the name for the Channel. This option is not required, and the name
// will default to a string version of the channels Key.
func (c Create) WithName(name string) Create { setName(c, name); return c }

// WithDataRate sets the data rate for the Channel. This option is required, and must be
// a non-zero value.
func (c Create) WithDataRate(dr telem.DataRate) Create { telem.SetDataRate(c, dr); return c }

// WithDataType sets the data type for the Channel. This option is required, and must be
// a non-zero value.
func (c Create) WithDataType(dt telem.DataType) Create { telem.SetDataType(c, dt); return c }

// WithTxn binds a transaction the query will be executed within. If the option is not
// set, the query will be executed directly against the Service database.
func (c Create) WithTxn(txn gorp.Txn) Create { gorp.SetTxn(c, txn); return c }

// Exec executes the query and returns the created Channel.
func (c Create) Exec(ctx context.Context) (Channel, error) {
	channels, err := c.ExecN(ctx, 1)
	if err != nil {
		return Channel{}, err
	}
	return channels[0], nil
}

// ExecN creates N channels using the same parameters.
func (c Create) ExecN(ctx context.Context, n int) ([]Channel, error) {
	channels, err := assembleFromQuery(c, n)
	if err != nil {
		return channels, err
	}
	return c.proxy.create(ctx, gorp.GetTxn(c, c.proxy.db), channels)
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
