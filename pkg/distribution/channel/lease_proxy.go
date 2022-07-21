package channel

import (
	"context"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/delta/pkg/distribution/proxy"
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/x/gorp"
)

type leaseProxy struct {
	cluster   aspen.Cluster
	db        *gorp.DB
	cesiumDB  cesium.DB
	transport CreateTransport
	router    proxy.BatchFactory[Channel]
	resources *ontology.Ontology
}

func newLeaseProxy(
	cluster aspen.Cluster,
	metadataDB *gorp.DB,
	cesiumDB cesium.DB,
	transport CreateTransport,
) *leaseProxy {
	p := &leaseProxy{
		cluster:   cluster,
		db:        metadataDB,
		cesiumDB:  cesiumDB,
		transport: transport,
		router:    proxy.NewBatchFactory[Channel](cluster.HostID()),
	}
	p.transport.Handle(p.handle)
	return p
}

func (lp *leaseProxy) handle(ctx context.Context, msg CreateRequest) (CreateRequest, error) {
	txn := lp.db.BeginTxn()
	channels, err := lp.create(ctx, txn, msg.Channels)
	if err != nil {
		return CreateRequest{}, err
	}
	return CreateRequest{Channels: channels}, txn.Commit()
}

func (lp *leaseProxy) create(
	ctx context.Context,
	txn gorp.Txn,
	channels []Channel,
) ([]Channel, error) {
	batch := lp.router.Batch(channels)
	oChannels := make([]Channel, 0, len(channels))
	for nodeID, entries := range batch.Remote {
		remoteChannels, err := lp.createRemote(ctx, nodeID, entries)
		if err != nil {
			return nil, err
		}
		oChannels = append(oChannels, remoteChannels...)
	}
	ch, err := lp.createLocal(txn, batch.Local)
	if err != nil {
		return oChannels, err
	}
	oChannels = append(oChannels, ch...)
	return oChannels, nil
}

func (lp *leaseProxy) createLocal(
	txn gorp.Txn,
	channels []Channel,
) ([]Channel, error) {
	for i, ch := range channels {
		key, err := lp.cesiumDB.CreateChannel(ch.Cesium)
		if err != nil {
			return nil, err
		}
		channels[i].Cesium.Key = key
	}
	// TODO: add transaction rollback to cesium db if this fails.
	if err := gorp.NewCreate[Key, Channel]().
		Entries(&channels).Exec(txn); err != nil {
		return nil, err
	}
	return channels, lp.maybeSetResources(txn, channels)
}

func (lp *leaseProxy) maybeSetResources(
	txn gorp.Txn,
	channels []Channel,
) error {
	if lp.resources != nil {
		w := lp.resources.NewWriter(txn)
		for _, channel := range channels {
			rtk := ResourceTypeKey(channel.Key())
			if err := w.DefineResource(rtk); err != nil {
				return err
			}
			if err := w.DefineRelationship(
				node.ResourceKey(channel.NodeID),
				rtk,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (lp *leaseProxy) createRemote(ctx context.Context,
	target aspen.NodeID, channels []Channel) ([]Channel, error) {
	addr, err := lp.cluster.Resolve(target)
	if err != nil {
		return nil, err
	}
	res, err := lp.transport.Send(ctx, addr, CreateRequest{Channels: channels})
	if err != nil {
		return nil, err
	}
	return res.Channels, nil
}
