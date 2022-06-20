package channel

import (
	"context"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/proxy"
	"github.com/arya-analytics/x/gorp"
)

type leaseProxy struct {
	cluster    aspen.Cluster
	metadataDB *gorp.DB
	cesiumDB   cesium.DB
	transport  CreateTransport
	router     proxy.Router[Channel]
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
		router:     proxy.NewRouter[Channel](cluster.HostID()),
	}
	p.transport.Handle(p.handle)
	return p
}

func (lp *leaseProxy) handle(ctx context.Context, msg CreateMessage) (CreateMessage, error) {
	channels, err := lp.create(ctx, msg.Channels)
	return CreateMessage{Channels: channels}, err
}

func (lp *leaseProxy) create(ctx context.Context, channels []Channel) ([]Channel, error) {
	local, remote := lp.router.Route(channels)
	oChannels := make([]Channel, 0, len(channels))
	for nodeID, batch := range remote {
		remoteChannels, err := lp.createRemote(ctx, nodeID, batch)
		if err != nil {
			return nil, err
		}
		oChannels = append(oChannels, remoteChannels...)
	}
	ch, err := lp.createLocal(local)
	if err != nil {
		return oChannels, err
	}
	oChannels = append(oChannels, ch...)
	return oChannels, nil
}

func (lp *leaseProxy) createLocal(channels []Channel) ([]Channel, error) {
	for i, ch := range channels {
		key, err := lp.cesiumDB.CreateChannel(ch.Cesium)
		if err != nil {
			return nil, err
		}
		channels[i].Cesium.Key = key
	}
	// TODO: add transaction rollback to cesium db if this fails.
	if err := gorp.NewCreate[Key, Channel]().
		Entries(&channels).Exec(lp.metadataDB); err != nil {
		return nil, err
	}
	return channels, nil
}

func (lp *leaseProxy) createRemote(ctx context.Context,
	target aspen.NodeID, channels []Channel) ([]Channel, error) {
	addr, err := lp.cluster.Resolve(target)
	if err != nil {
		return nil, err
	}
	res, err := lp.transport.Send(ctx, addr, CreateMessage{Channels: channels})
	if err != nil {
		return nil, err
	}
	return res.Channels, nil
}
