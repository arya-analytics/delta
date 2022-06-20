package segment

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/telem"
)

type Iterator interface {
	confluence.UnarySource[cesium.Segment]
	Next() bool
	Prev() bool
	First() bool
	Last() bool
	NextSpan(span telem.TimeSpan) bool
	PrevSpan(span telem.TimeSpan) bool
	NextRange(tr telem.TimeRange) bool
	SeekFirst() bool
	SeekLast() bool
	SeekLT(t telem.TimeStamp) bool
	SeekGE(t telem.TimeStamp) bool
	View() telem.TimeRange
	Exhaust()
	Error() error
	Close() error
}
