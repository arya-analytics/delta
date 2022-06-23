package iterator

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
)

// emitter translates iterator commands into requests and writes them to a stream.
type emitter struct {
	confluence.UnarySource[Request]
}

// Next emits a Next request to the stream.
func (e *emitter) Next() { e.emit(Request{Command: Next}) }

// Prev emits a Prev request to the stream.
func (e *emitter) Prev() { e.emit(Request{Command: Prev}) }

// First emits a First request to the stream.
func (e *emitter) First() { e.emit(Request{Command: First}) }

// Last emits a Last request to the stream.
func (e *emitter) Last() { e.emit(Request{Command: Last}) }

// NextSpan emits a NextSpan request to the stream.
func (e *emitter) NextSpan(span telem.TimeSpan) {
	e.emit(Request{Command: NextSpan, Span: span})
}

// PrevSpan emits a PrevSpan request to the stream.
func (e *emitter) PrevSpan(span telem.TimeSpan) {
	e.emit(Request{Command: PrevSpan, Span: span})
}

// NextRange emits a NextRange request to the stream.
func (e *emitter) NextRange(rng telem.TimeRange) {
	e.emit(Request{Command: NextRange, Range: rng})
}

// SeekFirst emits a SeekFirst request to the stream.
func (e *emitter) SeekFirst() { e.emit(Request{Command: SeekFirst}) }

// SeekLast emits a SeekLast request to the stream.
func (e *emitter) SeekLast() { e.emit(Request{Command: SeekLast}) }

// SeekLT emits a SeekLT request to the stream.
func (e *emitter) SeekLT(stamp telem.TimeStamp) {
	e.emit(Request{Command: SeekLT, Stamp: stamp})
}

// SeekGE emits a SeekGE request to the stream.
func (e *emitter) SeekGE(stamp telem.TimeStamp) {
	e.emit(Request{Command: SeekGE, Stamp: stamp})
}

// Close emits a Close request to the stream.
func (e *emitter) Close() { e.emit(Request{Command: Close}) }

// Exhaust emits an Exhaust request to the stream.
func (e *emitter) Exhaust() { e.emit(Request{Command: Exhaust}) }

func (e *emitter) emit(req Request) { e.Out.Inlet() <- req }

func executeRequest(iter cesium.StreamIterator, req Request) Response {
	switch req.Command {
	case Open:
		ack := newAck(false)
		ack.Error = errors.New(
			"[segment.iterator.serve] - Open command called multiple times",
		)
		return ack
	case Next:
		return newAck(iter.Next())
	case Prev:
		return newAck(iter.Prev())
	case First:
		return newAck(iter.First())
	case Last:
		return newAck(iter.Last())
	case NextSpan:
		return newAck(iter.NextSpan(req.Span))
	case PrevSpan:
		return newAck(iter.PrevSpan(req.Span))
	case NextRange:
		return newAck(iter.NextRange(req.Range))
	case SeekFirst:
		return newAck(iter.SeekFirst())
	case SeekLast:
		return newAck(iter.SeekLast())
	case SeekLT:
		return newAck(iter.SeekLT(req.Stamp))
	case SeekGE:
		return newAck(iter.SeekGE(req.Stamp))
	case Exhaust:
		iter.Exhaust()
		return Response{}
	case Close:
		err := iter.Close()
		ack := newAck(err == nil)
		ack.Error = err
		return ack
	default:
		ack := newAck(false)
		ack.Error = errors.New("[segment.iterator.serve] - unknown command")
		return ack
	}
}
