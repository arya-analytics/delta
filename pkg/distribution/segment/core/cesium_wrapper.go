package core

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
)

// CesiumWrapper wraps slices of cesium.Segment into slices of Segment by
// injecting the appropriate channel keys. All values in KeyMap should be loaded
// into the wrapper before calling Wrap for the first time.
type CesiumWrapper struct {
	KeyMap map[cesium.ChannelKey]channel.Key
}

// Wrap
func (cw *CesiumWrapper) Wrap(segments []cesium.Segment) []Segment {
	wrapped := make([]Segment, len(segments))
	for i, seg := range segments {
		key, ok := cw.KeyMap[seg.ChannelKey]
		if !ok {
			panic("[segment.iterator.serve] - channel key not found in keymap. bug.")
		}
		wrapped[i] = Segment{ChannelKey: key, Segment: seg}
	}
	return wrapped
}
