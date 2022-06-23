package core

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
)

type CesiumWrapper struct {
	KeyMap map[cesium.ChannelKey]channel.Key
}

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
