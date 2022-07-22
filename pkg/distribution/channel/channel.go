package channel

import (
	"encoding/binary"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/x/filter"
	"github.com/cockroachdb/errors"
	"strconv"
	"strings"
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

// Cesium returns a unique identifier for the Channel within the leaseholder node's
// cesium.DB. This value is NOT guaranteed to be unique across the entire cluster.
func (c Key) Cesium() cesium.ChannelKey {
	return cesium.ChannelKey(binary.LittleEndian.Uint16(c[4:6]))
}

// Lease implements the proxy.Entry interface.
func (c Key) Lease() aspen.NodeID { return c.NodeID() }

const strKeySep = "-"

// String returns the Key as a string in the format of "NodeID-CesiumKey", so
// a channel with a NodeID of 1 and a CesiumKey of 2 would return "1-2".
func (c Key) String() string {
	return strconv.Itoa(int(c.NodeID())) + strKeySep + strconv.Itoa(int(c.Cesium()))
}

// ParseKey parses a string representation of a Key into a Key. The key must be in
// the format outlined in Key.String().
func ParseKey(s string) (k Key, err error) {
	split := strings.Split(s, "-")
	if len(split) != 2 {
		return k, errors.New("[channel] - invalid key format")
	}
	nodeID, err := strconv.Atoi(split[0])
	if err != nil {
		return k, errors.Wrapf(err, "[channel] - invalid node id")
	}
	cesiumKey, err := strconv.Atoi(split[1])
	if err != nil {
		return k, errors.Wrapf(err, "[channel] - invalid cesium key")
	}
	return NewKey(aspen.NodeID(nodeID), cesium.ChannelKey(cesiumKey)), nil
}

// OntologyID returns a unique identifier for a Channel for use within a resource
// ontology.
func OntologyID(k Key) ontology.ID {
	return ontology.ID{Type: ontologyType, Key: k.String()}
}

// Keys extends []Key with a few convenience methods.
type Keys []Key

// ParseKeys parses a slice of strings into a slice of Keys.
func ParseKeys(keys []string) (Keys, error) {
	var err error
	k := make(Keys, len(keys))
	for i, key := range keys {
		k[i], err = ParseKey(key)
		if err != nil {
			return nil, err
		}
	}
	return k, nil
}

// Cesium calls Key.Cesium() on each key and returns a slice with the results.
func (k Keys) Cesium() []cesium.ChannelKey {
	keys := make([]cesium.ChannelKey, len(k))
	for i, key := range k {
		keys[i] = key.Cesium()
	}
	return keys
}

// UniqueNodeIDs returns a slice of all UNIQUE node IDs for the given keys.
func (k Keys) UniqueNodeIDs() (ids []node.ID) {
	for _, key := range k {
		if !filter.ElementOf(ids, key.NodeID()) {
			ids = append(ids, key.NodeID())
		}
	}
	return ids
}

// Strings returns the keys as a slice of strings.
func (k Keys) Strings() []string {
	s := make([]string, len(k))
	for i, key := range k {
		s[i] = key.String()
	}
	return s
}

// Channel is a collection is a container representing a collection of samples across
// a time range. The data within a channel typically arrives from a single source. This
// can be a physical sensor, software sensor, metric, event, or any other entity that
// emits regular, consistent, and time-ordered values.
//
// This Channel type (for the distribution layer) extends a cesium.DB's channel via
// composition to add fields necessary for cluster wide distribution.
//
// A Channel "belongs to" a specific Node. Because delta is oriented towards data collection
// close to the hardware, it's natural to assume a sensor writes to one and only device.
// For example, we may have a temperature sensor for a carbon fiber oven connected to a
// Linux box. The temperature sensor is a Channel that writes to Node residing on the
// Linux box.
//
// Data for a channel can only be written through the leaseholder. This helps solve a lot
// of consistency and atomicity issues.
type Channel struct {
	// Name is a human-readable name for the channel. This name does not have to be
	// unique.
	Name string
	// NodeID is the leaseholder node for the channel.
	NodeID node.ID
	// Cesium are properties of the channel necessary for cesium to perform
	// operations on it. See the cesium.Channel docs for mroe info on this.
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
