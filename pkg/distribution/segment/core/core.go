package core

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/x/query"
	"github.com/cockroachdb/errors"
)

type Segment struct {
	ChannelKey channel.Key
	Segment    cesium.Segment
}

func ValidateChannelKeys(ctx context.Context, svc *channel.Service, keys []channel.Key) error {
	if len(keys) == 0 {
		return errors.New("[segment] - no channels provided")
	}
	exists, err := svc.NewRetrieve().WhereKeys(keys...).Exists(ctx)
	if !exists {
		return errors.Wrapf(query.NotFound, "[segment] - channel keys %s not found", keys)
	}
	if err != nil {
		return errors.Wrap(err, "[segment] - failed to validate channel keys")
	}
	return nil
}
