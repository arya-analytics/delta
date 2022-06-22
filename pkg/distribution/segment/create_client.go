package segment

import (
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/shutdown"
)

type createClient struct {
	confluence.UnarySink[CreateRequest]
	confluence.UnarySource[CreateResponse]
	client CreateClient
}

func (cc *createClient) Flow(ctx confluence.Context) {
	ctx.Shutdown.Go(func(sig chan shutdown.Signal) error {
		for {
			select {
			case <-sig:
				if err := cc.client.CloseSend(); err != nil {
					return err
				}
			default:
				res, err := cc.client.Receive()
				if err != nil {
					return err
				}
				cc.UnarySource.Out.Inlet() <- res
			}
		}
	})
	ctx.Shutdown.Go(func(sig chan shutdown.Signal) error {
		for {
			select {
			case <-sig:
				return nil
			case req := <-cc.UnarySink.In.Outlet():
				if err := cc.client.Send(req); err != nil {
					cc.UnarySource.Out.Inlet() <- CreateResponse{Error: err}
				}
			}
		}
	})
}

type createServer struct {
}

type createClientFactory struct {
	transport CreateServer
}

func (ccf *createClientFactory) new(target address.Address) error {
	return &createClient{}
}
