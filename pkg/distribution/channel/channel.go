package channel

import (
	"encoding/binary"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
)

// Key represents a unique identifier for a Channel. This value is guaranteed to be
// unique across the entire cluster. It is composed of a uint32 ID representing the
// node holding the lease on the channel, and a uint16 key representing a unique
// identifier for the channel on the node's cesium.DB.
type Key [6]byte

// NewKey generates a new Key from the provided components.
func NewKey(nodeID aspen.NodeID, cesiumKey cesium.ChannelKey) (key Key) {
	binary.LittleEndian.PutUint32(key[0:4], uint32(nodeID))
	binary.LittleEndian.PutUint16(key[4:6], uint16(cesiumKey))
	return key
}

// NodeID returns the id of the node embedded in the key. This node is the leaseholder
// node for the Channel.
func (c Key) NodeID() aspen.NodeID { return aspen.NodeID(binary.LittleEndian.Uint32(c[0:4])) }

// CesiumKey returns a unique identifier for the Channel within the leaseholder node's
// cesium.DB. This value is NOT guaranteed tobe unique across the entire cluster.
func (c Key) CesiumKey() cesium.ChannelKey {
	return cesium.ChannelKey(binary.LittleEndian.Uint16(c[4:6]))
}

// Lease implements the proxy.Route interface.
func (c Key) Lease() aspen.NodeID { return c.NodeID() }

type Channel struct {
	Name   string
	NodeID aspen.NodeID
	Cesium cesium.Channel
}

// Key returns the key for the Channel.
func (c Channel) Key() Key { return NewKey(c.NodeID, c.Cesium.Key) }

// GorpKey implements the gorp.Entry interface.
func (c Channel) GorpKey() Key { return c.Key() }

func (c Channel) SetOptions() []interface{} { return []interface{}{c.Lease()} }

// Lease implements the proxy.Route interface.
func (c Channel) Lease() aspen.NodeID { return c.NodeID }
