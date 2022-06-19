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

var _ = Describe("Create", func() {
	var (
		services  map[aspen.NodeID]*channel.Service
		aspenDBs  map[aspen.NodeID]aspen.DB
		cesiumDBs map[aspen.NodeID]cesium.DB
		log       *zap.Logger
	)
	BeforeEach(func() {
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
		services[db1.HostID()] = channel.New(db1, gorp.Wrap(db1), cdb1, net.Route(""))

		db2, err := aspenBuilder.New()
		Expect(err).ToNot(HaveOccurred())
		aspenDBs[db2.HostID()] = db2
		cdb2, err := cesium.Open("", cesium.MemBacked(), cesium.WithLogger(log))
		Expect(err).ToNot(HaveOccurred())
		cesiumDBs[db2.HostID()] = cdb2
		services[db2.HostID()] = channel.New(db2, gorp.Wrap(db2), cdb2, net.Route(""))
	})
	Context("Node is local", func() {
		It("Should create the channel without error", func() {
			ch, err := services[1].NewCreate().
				WithDataRate(5 * cesium.Hz).
				WithDataType(cesium.Float64).
				WithName("SG01").
				WithNodeID(1).
				Exec(ctx)
			Expect(err).To(BeNil())
			Expect(ch.Key().NodeID()).To(Equal(aspen.NodeID(1)))
			Expect(ch.Key().CesiumKey()).To(Equal(cesium.ChannelKey(1)))
		})
	})
	Context("Node is remote", func() {
		It("Should create the channel without error", func() {
			ch, err := services[1].NewCreate().
				WithDataRate(5 * cesium.Hz).
				WithDataType(cesium.Float64).
				WithName("SG01").
				WithNodeID(2).
				Exec(ctx)
			Expect(err).To(BeNil())
			Expect(ch.Key().NodeID()).To(Equal(aspen.NodeID(2)))
			Expect(ch.Key().CesiumKey()).To(Equal(cesium.ChannelKey(1)))
		})
		It("Should assign a cesium key of 1 to the first channels on each node",
			func() {
				ch, err := services[1].NewCreate().
					WithDataRate(5 * cesium.Hz).
					WithDataType(cesium.Float64).
					WithName("SG01").
					WithNodeID(1).
					Exec(ctx)
				Expect(err).To(BeNil())
				Expect(ch.Key().NodeID()).To(Equal(aspen.NodeID(1)))
				Expect(ch.Key().CesiumKey()).To(Equal(cesium.ChannelKey(1)))
				ch2, err := services[1].NewCreate().
					WithDataRate(5 * cesium.Hz).
					WithDataType(cesium.Float64).
					WithName("SG01").
					WithNodeID(2).
					Exec(ctx)
				Expect(err).To(BeNil())
				Expect(ch2.Key().NodeID()).To(Equal(aspen.NodeID(2)))
				Expect(ch2.Key().CesiumKey()).To(Equal(cesium.ChannelKey(1)))

			})
	})
})
