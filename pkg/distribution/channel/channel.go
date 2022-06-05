package channel

import (
	"encoding/binary"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
)

type Key [6]byte

func (c Key) NodeID() aspen.NodeID { return aspen.NodeID(binary.LittleEndian.Uint32(c[0:4])) }

func (c Key) ChannelKey() cesium.ChannelKey {
	return cesium.ChannelKey(binary.LittleEndian.Uint16(c[4:6]))
}

type Channel struct {
	NodeID aspen.NodeID
	Cesium cesium.Channel
}

func (c Channel) Key() Key {
	var b [6]byte
	binary.LittleEndian.PutUint32(b[0:4], uint32(c.NodeID))
	binary.LittleEndian.PutUint16(b[4:6], uint16(c.Cesium.Key))
	return b
}

func (c Channel) SetOptions() []interface{} { return []interface{}{c.NodeID} }
