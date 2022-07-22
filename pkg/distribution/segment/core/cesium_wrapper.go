package core

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/node"
)

// CesiumWrapper wraps slices of cesium.Segment into slices of Segment by
// adding the appropriate host information.
type CesiumWrapper struct {
	Host node.ID
}

// Wrap converts a slice of cesium.Segment into a slice of Segment.
func (cw *CesiumWrapper) Wrap(segments []cesium.Segment) []Segment {
	wrapped := make([]Segment, len(segments))
	for i, seg := range segments {
		wrapped[i] = Segment{
			ChannelKey: channel.NewKey(cw.Host, seg.ChannelKey),
			Segment:    seg,
		}
	}
	return wrapped
}
