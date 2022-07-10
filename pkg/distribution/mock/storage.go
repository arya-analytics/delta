package mock

import (
	"github.com/arya-analytics/aspen"
	aspenmock "github.com/arya-analytics/aspen/mock"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/x/errutil"
	"go.uber.org/zap"
)

type StorageBuilder struct {
	aspenBuilder *aspenmock.Builder
	Stores       map[aspen.NodeID]Store
}

type Store struct {
	Aspen  aspen.DB
	Cesium cesium.DB
}

func NewStorage() *StorageBuilder {
	return &StorageBuilder{
		Stores:       make(map[aspen.NodeID]Store),
		aspenBuilder: aspenmock.NewMemBuilder(),
	}
}

func (sb *StorageBuilder) New(logger *zap.Logger) (Store, error) {
	store := Store{}
	var err error
	store.Aspen, err = sb.aspenBuilder.New(aspen.WithLogger(logger.Sugar()))
	if err != nil {
		return store, err
	}
	store.Cesium, err = cesium.Open("", cesium.MemBacked(), cesium.WithLogger(logger))
	if err != nil {
		return store, err
	}
	sb.Stores[store.Aspen.HostID()] = store
	return store, nil
}

func (sb *StorageBuilder) Close() error {
	c := errutil.NewCatchSimple(errutil.WithAggregation())
	for _, store := range sb.Stores {
		c.Exec(store.Aspen.Close)
		c.Exec(store.Cesium.Close)
	}
	return nil
}
