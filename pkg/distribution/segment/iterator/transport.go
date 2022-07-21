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
	Valid
	Error
	Close
	Exhaust
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
	AckResponse ResponseVariant = iota + 1
	DataResponse
)

type Response struct {
	Variant  ResponseVariant
	NodeID   node.ID
	Ack      bool
	Command  Command
	Error    error
	Segments []core.Segment
}

func newAck(host node.ID, cmd Command, ok bool) Response {
	return Response{Variant: AckResponse, Ack: ok, Command: cmd, NodeID: host}
}

type (
	Server    = transport.StreamServer[Request, Response]
	Client    = transport.StreamClient[Request, Response]
	Transport = transport.Stream[Request, Response]
)
