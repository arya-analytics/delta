package writer_test

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/cesium/testutil/seg"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/telem"
	tmock "github.com/arya-analytics/x/transport/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"time"
)

var _ = Describe("Remote", Ordered, func() {
	var (
		log       *zap.Logger
		net       *tmock.Network[writer.Request, writer.Response]
		builder   *mock.StorageBuilder
		w         writer.Writer
		factory   seg.SequentialFactory
		wrapper   *core.CesiumWrapper
		keys      channel.Keys
		newWriter func() (writer.Writer, error)
		channels  []channel.Channel
	)
	BeforeAll(func() {
		log = zap.NewNop()
		builder = mock.NewStorage()
		dataFactory := &seg.RandomFloat64Factory{Cache: true}
		net = tmock.NewNetwork[writer.Request, writer.Response]()
		channelNet := tmock.NewNetwork[channel.CreateRequest, channel.CreateRequest]()

		node1Addr := address.Address("localhost:0")
		node2Addr := address.Address("localhost:1")
		node3Addr := address.Address("localhost:2")

		store1, err := builder.New(log)
		Expect(err).ToNot(HaveOccurred())
		node1Transport := net.RouteStream(node1Addr, 0)
		writer.NewServer(store1.Cesium, store1.Aspen.HostID(), node1Transport)

		store2, err := builder.New(log)
		Expect(err).ToNot(HaveOccurred())
		node2Transport := net.RouteStream(node2Addr, 0)
		writer.NewServer(store2.Cesium, store2.Aspen.HostID(), node2Transport)

		store3, err := builder.New(log)
		Expect(err).ToNot(HaveOccurred())
		node3Transport := net.RouteStream(node3Addr, 0)
		writer.NewServer(store3.Cesium, store3.Aspen.HostID(), node3Transport)

		store1ChannelSvc := channel.New(
			store1.Aspen,
			gorp.Wrap(store1.Aspen),
			store1.Cesium,
			channelNet.RouteUnary(node1Addr),
		)

		store2ChannelSvc := channel.New(
			store2.Aspen,
			gorp.Wrap(store2.Aspen),
			store2.Cesium,
			channelNet.RouteUnary(node2Addr),
		)

		store3ChannelSvc := channel.New(
			store3.Aspen,
			gorp.Wrap(store3.Aspen),
			store3.Cesium,
			channelNet.RouteUnary(node3Addr),
		)

		dr := 1 * telem.Hz
		store1Channels, err := store1ChannelSvc.NewCreate().
			WithName("SG02").
			WithDataRate(dr).
			WithDataType(telem.Float64).
			WithNodeID(1).
			ExecN(ctx, 1)
		Expect(err).ToNot(HaveOccurred())
		channels = append(channels, store1Channels...)

		store2Channels, err := store2ChannelSvc.NewCreate().
			WithName("SG02").
			WithDataRate(dr).
			WithDataType(telem.Float64).
			WithNodeID(2).
			ExecN(ctx, 1)
		Expect(err).ToNot(HaveOccurred())

		channels = append(channels, store2Channels...)
		var cesiumChannels []cesium.Channel
		for _, c := range channels {
			cesiumChannels = append(cesiumChannels, c.Cesium)
		}

		factory = seg.NewSequentialFactory(dataFactory, 10*telem.Second, cesiumChannels...)
		wrapper = &core.CesiumWrapper{KeyMap: map[cesium.ChannelKey]channel.Key{
			channels[0].Cesium.Key: channels[0].Key(),
			channels[1].Cesium.Key: channels[1].Key(),
		}}

		keys = channel.Keys{channels[0].Key(), channels[1].Key()}

		node1Transport = net.RouteStream("", 0)

		time.Sleep(150 * time.Millisecond)

		newWriter = func() (writer.Writer, error) {
			return writer.New(
				ctx,
				store3.Cesium,
				store3ChannelSvc,
				store3.Aspen,
				node3Transport,
				keys,
			)
		}
	})
	BeforeEach(func() {
		var err error
		w, err = newWriter()
		Expect(err).ToNot(HaveOccurred())
	})
	AfterAll(func() { Expect(builder.Close()).To(Succeed()) })
	Context("Behavioral Accuracy", func() {
		It("should write the segment to disk", func() {
			seg := wrapper.Wrap(factory.NextN(1))
			seg[0].ChannelKey = channels[0].Key()
			seg[1].ChannelKey = channels[1].Key()
			w.Requests() <- writer.Request{Segments: seg}
			close(w.Requests())
			for res := range w.Responses() {
				Expect(res.Error).ToNot(HaveOccurred())
			}
			Expect(w.Close()).To(Succeed())
		})
	})
})
