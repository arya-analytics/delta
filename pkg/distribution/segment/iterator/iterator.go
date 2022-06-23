package iterator

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/proxy"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/query"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
)

type Iterator interface {
	confluence.Source[cesium.Segment]
	Next() bool
	Prev() bool
	First() bool
	Last() bool
	NextSpan(span telem.TimeSpan) bool
	PrevSpan(span telem.TimeSpan) bool
	NextRange(tr telem.TimeRange) bool
	SeekFirst() bool
	SeekLast() bool
	SeekLT(t telem.TimeStamp) bool
	SeekGE(t telem.TimeStamp) bool
	View() telem.TimeRange
	Exhaust()
	Error() error
	Close() error
}

func NewIterator(
	ctx context.Context,
	svc *channel.Service,
	rng telem.TimeRange,
	keys ...channel.Key,
) (Iterator, error) {

	// First we need to check if all the channels exists and are retrievable in the
	// database.
	if err := validateChannelKeys(ctx, svc, keys); err != nil {
		return nil, err
	}

	// Next we determine IDs of all the hosts we need to open client iterators on.
	local, remote := proxy.NewBatchFactory[channel.Key](svc.HostID()).Batch(keys)

}

func validateChannelKeys(ctx context.Context, svc *channel.Service, keys []channel.Key) error {
	exists, err := svc.NewRetrieve().WhereKeys(keys...).Exists(ctx)
	if !exists {
		return errors.Wrap(query.NotFound, "[segment.iterator] - channel keys not found")
	}
	if err != nil {
		return errors.Wrap(err, "[segment.iterator] - failed to validate channel keys")
	}
	return nil
}
