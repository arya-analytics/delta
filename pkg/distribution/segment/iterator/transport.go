package iterator

import (
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/telem"
	"github.com/arya-analytics/x/transport"
)

type Command uint8

const (
	Open Command = iota
	Next
	Prev
	First
	Last
	NextSpan
	PrevSpan
	NextRange
	SeekFirst
	SeekLast
	SeekLT
	SeekGE
	Exhaust
	Error
	Close
	EOF
)

type Request struct {
	Command Command
	Span    telem.TimeSpan
	Range   telem.TimeRange
	Stamp   telem.TimeStamp
	Keys    channel.Keys
}

type ResponseVariant uint8

const (
	ResponseVariantAck ResponseVariant = iota + 1
	ResponseVariantData
)

type Response struct {
	Variant  ResponseVariant
	NodeID   node.ID
	Ack      bool
	Command  Command
	Segments []core.Segment
	Error    error
}

func newAck(host node.ID, cmd Command, ok bool) Response {
	return Response{Variant: ResponseVariantAck, Ack: ok, Command: cmd, NodeID: host}
}

type (
	Server    = transport.StreamServer[Request, Response]
	Client    = transport.StreamClient[Request, Response]
	Transport = transport.Stream[Request, Response]
)
