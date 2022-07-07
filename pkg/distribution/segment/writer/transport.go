package writer

import (
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/transport"
)

type Request struct {
	OpenKeys channel.Keys
	Segments []core.Segment
}

type Response struct {
	Error error
}

type (
	Server    = transport.StreamServer[Request, Response]
	Client    = transport.StreamClient[Request, Response]
	Transport = transport.Stream[Request, Response]
)
