package channel

import "github.com/arya-analytics/x/transport"

type CreateTransport = transport.Unary[CreateMessage, CreateMessage]

type CreateMessage struct {
	Channels []Channel
}
