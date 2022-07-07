package segment

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
)

type Service struct {
	channel  *channel.Service
	cesiumDB cesium.DB
}
