package iterator

import (
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/telem"
)

type client struct {
	target    address.Address
	requests  confluence.Sink[Request]
	responses confluence.Source[Response]
}

func (c *client) Flow(ctx signal.Context) {
	c.requests.Flow(ctx)
	c.responses.Flow(ctx)
}

func (c *client) OutTo(inlets ...confluence.Inlet[Response]) { c.responses.OutTo(inlets...) }

func (c *client) InFrom(outlets ...confluence.Outlet[Request]) { c.requests.InFrom(outlets...) }

func newClient(
	ctx signal.Context,
	transport Transport,
	target address.Address,
	keys channel.Keys,
	rng telem.TimeRange,
) (confluence.Translator[Request, Response], error) {
	stream, err := transport.Stream(ctx, target)
	if err != nil {
		return nil, err
	}

	if err := stream.Send(Request{
		Command: Open,
		Keys:    keys,
		Range:   rng,
	}); err != nil {
		return nil, err
	}

	sender := &confluence.Sender[Request]{Sender: stream}
	receiver := &confluence.Receiver[Response]{Receiver: stream}

	return &client{requests: sender, responses: receiver}, nil
}
