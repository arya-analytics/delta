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

// NodeID returns the id of the node embedded in the key. This node is the leaseholder
// node for the Channel.
func (c Key) NodeID() aspen.NodeID { return aspen.NodeID(binary.LittleEndian.Uint32(c[0:4])) }

// ChannelKey returns a unique identifier for the Channel within the leaseholder node's
// cesium.DB. This value is NOT guaranteed tobe unique across the entire cluster.
func (c Key) ChannelKey() cesium.ChannelKey {
	return cesium.ChannelKey(binary.LittleEndian.Uint16(c[4:6]))
}

type Channel struct {
	Name   string
	NodeID aspen.NodeID
	Cesium cesium.Channel
}

// Key returns the key for the Channel.
func (c Channel) Key() Key {
	var b [6]byte
	binary.LittleEndian.PutUint32(b[0:4], uint32(c.NodeID))
	binary.LittleEndian.PutUint16(b[4:6], uint16(c.Cesium.Key))
	return b
}

// GorpKey implements the gorp.Entry interface.
func (c Channel) GorpKey() Key { return c.Key() }

func (c Channel) SetOptions() []interface{} { return []interface{}{c.NodeID} }
