package channel_test

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/mock"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/telem"
	tmock "github.com/arya-analytics/x/transport/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"time"
)

var _ = Describe("getAttributes", Ordered, func() {
	var (
		services map[aspen.NodeID]*channel.Service
		builder  *mock.StorageBuilder
		log      *zap.Logger
	)
	BeforeAll(func() {
		log = zap.NewNop()
		services = make(map[aspen.NodeID]*channel.Service)
		net := tmock.NewNetwork[channel.CreateRequest, channel.CreateRequest]()
		builder = mock.NewStorage()
		store1, err := builder.New(log)
		Expect(err).To(BeNil())
		services[1] = channel.New(
			store1.Aspen,
			gorp.Wrap(store1.Aspen),
			store1.Cesium,
			net.RouteUnary(""),
		)
		store2, err := builder.New(log)
		Expect(err).To(BeNil())
		services[2] = channel.New(
			store2.Aspen,
			gorp.Wrap(store2.Aspen),
			store2.Cesium,
			net.RouteUnary(""),
		)
	})
	AfterAll(func() { Expect(builder.Close()).To(Succeed()) })
	It("Should correctly retrieve a set of channels", func() {
		created, err := services[1].NewCreate().
			WithName("SG02").
			WithDataRate(25*telem.KHz).
			WithDataType(telem.Float32).
			WithNodeID(1).
			ExecN(ctx, 10)
		Expect(err).ToNot(HaveOccurred())

		var resChannels []channel.Channel

		err = services[1].
			NewRetrieve().
			WhereNodeID(1).
			Entries(&resChannels).
			Exec(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(resChannels).To(HaveLen(len(created)))

		// Wait for the operations to propagate to another node.
		time.Sleep(60 * time.Millisecond)

		var resChannelsTwo []channel.Channel

		err = services[2].
			NewRetrieve().
			WhereNodeID(1).
			Entries(&resChannelsTwo).
			Exec(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(resChannelsTwo).To(HaveLen(len(created)))
	})
})
