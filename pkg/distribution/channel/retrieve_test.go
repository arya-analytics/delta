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
	"time"
)

var _ = Describe("Retrieve", Ordered, func() {
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
		services[db1.HostID()] = channel.New(db1, gorp.Wrap(db1), cdb1, net.Route(""))

		db2, err := aspenBuilder.New()
		Expect(err).ToNot(HaveOccurred())
		aspenDBs[db2.HostID()] = db2
		cdb2, err := cesium.Open("", cesium.MemBacked(), cesium.WithLogger(log))
		Expect(err).ToNot(HaveOccurred())
		cesiumDBs[db2.HostID()] = cdb2
		services[db2.HostID()] = channel.New(db2, gorp.Wrap(db2), cdb2, net.Route(""))
	})
	It("Should correctly retrieve a set of channels", func() {
		created, err := services[1].NewCreate().
			WithName("SG02").
			WithDataRate(25*cesium.KHz).
			WithDataType(cesium.Float32).
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
