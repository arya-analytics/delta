package segment

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
)

type Segment struct {
	ChannelKey channel.Key
	Cesium     cesium.Segment
}
