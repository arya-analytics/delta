package segment

import (
	"github.com/arya-analytics/x/telem"
	"github.com/arya-analytics/x/transport"
)

// |||||| CREATE ||||||

type CreateRequest struct {
	Segments []Segment
}

type CreateResponse struct {
	Error error
}

type (
	CreateServer = transport.StreamServer[CreateRequest, CreateResponse]
	CreateClient = transport.StreamClient[CreateRequest, CreateResponse]
)

// |||||| RETRIEVE ||||||

type IteratorCommand byte

const (
	Next IteratorCommand = iota
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
	View
	Exhaust
	Error
	Close
)

type RetrieveRequest struct {
	Command IteratorCommand
	Span    telem.TimeSpan
	Range   telem.TimeRange
	Stamp   telem.TimeStamp
}

type RetrieveResponse struct {
	Segments []Segment
	Error    error
}

type (
	RetrieveServer = transport.Stream[RetrieveRequest, RetrieveResponse]
)
