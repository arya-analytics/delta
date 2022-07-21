package channel

import "github.com/arya-analytics/x/transport"

type CreateTransport = transport.Unary[CreateRequest, CreateRequest]

type CreateRequest struct {
	Channels []Channel
}
