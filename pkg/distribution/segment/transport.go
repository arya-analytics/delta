package segment

import (
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/signal"
)

type Transport interface {
	Configure(ctx signal.Context, addr address.Address) error
	Iterator() iterator.Transport
	Writer() writer.Transport
}
