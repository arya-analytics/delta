package channel

import (
	"encoding/binary"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/x/filter"
	"github.com/cockroachdb/errors"
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

func ParseKey(s string) (k Key, err error) {
	b := []byte(s)
	if len(b) != len(k) {
		return k, errors.New("[channel.ID] - invalid length")
	}
	copy(k[:], b)
	return k, nil
}

// NodeID returns the id of the node embedded in the key. This node is the leaseholder
// node for the Channel.
func (c Key) NodeID() aspen.NodeID { return aspen.NodeID(binary.LittleEndian.Uint32(c[0:4])) }

// Cesium returns a unique identifier for the Channel within the leaseholder node's
// cesium.DB. This value is NOT guaranteed tobe unique across the entire cluster.
func (c Key) Cesium() cesium.ChannelKey {
	return cesium.ChannelKey(binary.LittleEndian.Uint16(c[4:6]))
}

// Lease implements the proxy.RouteUnary interface.
func (c Key) Lease() aspen.NodeID { return c.NodeID() }

func (c Key) String() string { return string(c[:]) }

func ResourceTypeKey(k Key) ontology.ID {
	return ontology.ID{Type: ontologyType, Key: k.String()}
}

type Keys []Key

func (k Keys) Cesium() []cesium.ChannelKey {
	keys := make([]cesium.ChannelKey, len(k))
	for i, key := range k {
		keys[i] = key.Cesium()
	}
	return keys
}

func (k Keys) CesiumMap() map[cesium.ChannelKey]Key {
	m := make(map[cesium.ChannelKey]Key)
	for _, key := range k {
		m[key.Cesium()] = key
	}
	return m
}

// Nodes returns a slice of all unique node IDs of Keys.
func (k Keys) Nodes() (ids []node.ID) {
	for _, key := range k {
		if !filter.ElementOf(ids, key.NodeID()) {
			ids = append(ids, key.NodeID())
		}
	}
	return ids
}

type Channel struct {
	Name   string
	NodeID node.ID
	Cesium cesium.Channel
}

// Key returns the key for the Channel.
func (c Channel) Key() Key { return NewKey(c.NodeID, c.Cesium.Key) }

// GorpKey implements the gorp.Entry interface.
func (c Channel) GorpKey() Key { return c.Key() }

// SetOptions implements the gorp.Entry interface. Returns a set of options that
// tell an aspen.DB to properly lease the Channel to the node it will be recording data
// from.
func (c Channel) SetOptions() []interface{} { return []interface{}{c.Lease()} }

// Lease implements the proxy.RouteUnary interface.
func (c Channel) Lease() aspen.NodeID { return c.NodeID }
