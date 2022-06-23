package channel_test

import (
	"github.com/arya-analytics/aspen"
	aspenmock "github.com/arya-analytics/aspen/mock"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/x/gorp"
	tmock "github.com/arya-analytics/x/transport/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Create", Ordered, func() {
	var (
		services  map[aspen.NodeID]*channel.Service
		aspenDBs  map[aspen.NodeID]aspen.DB
		cesiumDBs map[aspen.NodeID]cesium.DB
		log       *zap.Logger
	)
	BeforeAll(func() {
		log = zap.NewNop()
		services = make(map[aspen.NodeID]*channel.Service)
		aspenDBs = make(map[aspen.NodeID]aspen.DB)
		cesiumDBs = make(map[aspen.NodeID]cesium.DB)
		net := tmock.NewNetwork[channel.CreateMessage, channel.CreateMessage]()
		aspenBuilder := aspenmock.NewMemBuilder(aspen.WithLogger(log.Sugar()))
		db1, err := aspenBuilder.New()
		Expect(err).ToNot(HaveOccurred())
		aspenDBs[db1.HostID()] = db1
		cdb1, err := cesium.Open("", cesium.MemBacked(), cesium.WithLogger(log))
		Expect(err).ToNot(HaveOccurred())
		cesiumDBs[db1.HostID()] = cdb1
		services[db1.HostID()] = channel.New(db1, gorp.Wrap(db1), cdb1, net.RouteUnary(""))

		db2, err := aspenBuilder.New()
		Expect(err).ToNot(HaveOccurred())
		aspenDBs[db2.HostID()] = db2
		cdb2, err := cesium.Open("", cesium.MemBacked(), cesium.WithLogger(log))
		Expect(err).ToNot(HaveOccurred())
		cesiumDBs[db2.HostID()] = cdb2
		services[db2.HostID()] = channel.New(db2, gorp.Wrap(db2), cdb2, net.RouteUnary(""))
	})
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
				channels, err := cesiumDBs[1].RetrieveChannel(ch.Key().Cesium())
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
				channels, err := cesiumDBs[2].RetrieveChannel(ch.Key().Cesium())
				Expect(err).ToNot(HaveOccurred())
				Expect(channels).To(HaveLen(1))
				cesiumCH := channels[0]
				Expect(cesiumCH.Key).To(Equal(ch.Key().Cesium()))
				Expect(cesiumCH.DataType).To(Equal(cesium.Float64))
				Expect(cesiumCH.DataRate).To(Equal(5 * cesium.Hz))
			})
			It("Should not create the channel on another node's ceisum DB", func() {
				channels, err := cesiumDBs[1].RetrieveChannel(ch.Key().Cesium())
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
