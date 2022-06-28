package channel_test

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/mock"
	"github.com/arya-analytics/x/gorp"
	tmock "github.com/arya-analytics/x/transport/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Create", Ordered, func() {
	var (
		services map[aspen.NodeID]*channel.Service
		builder  *mock.StorageBuilder
		log      *zap.Logger
	)
	BeforeAll(func() {
		log = zap.NewNop()
		services = make(map[aspen.NodeID]*channel.Service)
		net := tmock.NewNetwork[channel.CreateMessage, channel.CreateMessage]()
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
	Context("Single Channel", func() {
		var (
			channelLeaseNodeID aspen.NodeID
			ch                 channel.Channel
		)
		JustBeforeEach(func() {
			var err error
			ch, err = services[1].NewCreate().
				WithDataRate(5 * cesium.Hz).
				WithDataType(cesium.Float64).
				WithName("SG01").
				WithNodeID(channelLeaseNodeID).
				Exec(ctx)
			Expect(err).ToNot(HaveOccurred())
		})
		Context("Node is local", func() {
			BeforeEach(func() { channelLeaseNodeID = 1 })
			It("Should create the channel without error", func() {
				Expect(ch.Key().NodeID()).To(Equal(aspen.NodeID(1)))
				Expect(ch.Key().Cesium()).To(Equal(cesium.ChannelKey(1)))
			})
			It("Should create the channel in the cesium DB", func() {
				channels, err := builder.Stores[1].Cesium.RetrieveChannel(ch.Key().Cesium())
				Expect(err).ToNot(HaveOccurred())
				Expect(channels).To(HaveLen(1))
				cesiumCH := channels[0]
				Expect(cesiumCH.Key).To(Equal(ch.Key().Cesium()))
				Expect(cesiumCH.DataType).To(Equal(cesium.Float64))
				Expect(cesiumCH.DataRate).To(Equal(5 * cesium.Hz))
			})
		})
		Context("Node is remote", func() {
			BeforeEach(func() { channelLeaseNodeID = 2 })
			It("Should create the channel without error", func() {
				Expect(ch.Key().NodeID()).To(Equal(aspen.NodeID(2)))
				Expect(ch.Key().Cesium()).To(Equal(cesium.ChannelKey(1)))
			})
			It("Should create the channel in the cesium DB", func() {
				channels, err := builder.Stores[2].Cesium.RetrieveChannel(ch.Key().Cesium())
				Expect(err).ToNot(HaveOccurred())
				Expect(channels).To(HaveLen(1))
				cesiumCH := channels[0]
				Expect(cesiumCH.Key).To(Equal(ch.Key().Cesium()))
				Expect(cesiumCH.DataType).To(Equal(cesium.Float64))
				Expect(cesiumCH.DataRate).To(Equal(5 * cesium.Hz))
			})
			It("Should not create the channel on another node's ceisum DB", func() {
				channels, err := builder.Stores[1].Cesium.RetrieveChannel(ch.Key().Cesium())
				Expect(err).ToNot(HaveOccurred())
				Expect(channels).To(HaveLen(0))
			})
			It("Should assign a sequential key to the channels on each node",
				func() {
					ch2, err := services[1].NewCreate().
						WithDataRate(5 * cesium.Hz).
						WithDataType(cesium.Float64).
						WithName("SG01").
						WithNodeID(1).
						Exec(ctx)
					Expect(err).To(BeNil())
					Expect(ch2.Key().NodeID()).To(Equal(aspen.NodeID(1)))
					Expect(ch2.Key().Cesium()).To(Equal(cesium.ChannelKey(3)))
				})
		})

	})
})
