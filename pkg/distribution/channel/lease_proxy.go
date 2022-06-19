package channel

import (
	"context"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/x/gorp"
)

type leaseProxy struct {
	cluster    aspen.Cluster
	metadataDB *gorp.DB
	cesiumDB   cesium.DB
	transport  CreateTransport
}

func newLeaseProxy(
	cluster aspen.Cluster,
	metadataDB *gorp.DB,
	cesiumDB cesium.DB,
	transport CreateTransport,
) *leaseProxy {
	p := &leaseProxy{
		cluster:    cluster,
		metadataDB: metadataDB,
		cesiumDB:   cesiumDB,
		transport:  transport,
	}
	p.transport.Handle(p.handle)
	return p
}

func (p *leaseProxy) handle(ctx context.Context, msg CreateMessage) (CreateMessage, error) {
	channels, err := p.create(ctx, msg.Channels)
	return CreateMessage{Channels: channels}, err
}

func (p *leaseProxy) create(ctx context.Context, channels []Channel) ([]Channel, error) {
	batched := BatchByNodeID(channels)
	oChannels := make([]Channel, 0, len(batched))
	for nodeID, batch := range batched {
		if p.cluster.HostID() != nodeID {
			addr, err := p.cluster.Resolve(nodeID)
			if err != nil {
				return nil, err
			}
			res, err := p.transport.Send(ctx, addr, CreateMessage{Channels: batch})
			if err != nil {
				return nil, err
			}
			oChannels = append(oChannels, res.Channels...)
		} else {
			for i, ch := range channels {
				key, err := p.cesiumDB.CreateChannel(ch.Cesium)
				if err != nil {
					return nil, err
				}
				channels[i].Cesium.Key = key
			}
			// TODO: add transaction rollback to cesium db if this fails.
			if err := gorp.NewCreate[Key, Channel]().
				Entries(&channels).Exec(p.metadataDB); err != nil {
				return nil, err
			}
			oChannels = append(oChannels, channels...)
		}
	}
	return oChannels, nil
}
