package writer_test

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/cesium/testutil/seg"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/telem"
	tmock "github.com/arya-analytics/x/transport/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"

	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
)

var _ = Describe("Local", Ordered, func() {
	var (
		log       *zap.Logger
		net       *tmock.Network[writer.Request, writer.Response]
		w         writer.Writer
		builder   *mock.StorageBuilder
		requests  confluence.Inlet[writer.Request]
		responses confluence.Outlet[writer.Response]
		factory   seg.SequentialFactory
		wrapper   *core.CesiumWrapper
	)
	BeforeAll(func() {
		log = zap.NewNop()
		builder = mock.NewStorage()
		dataFactory := &seg.RandomFloat64Factory{Cache: true}

		net = tmock.NewNetwork[writer.Request, writer.Response]()

		channelNet := tmock.NewNetwork[channel.CreateMessage, channel.CreateMessage]()

		store1, err := builder.New(log)
		Expect(err).ToNot(HaveOccurred())

		channelSvc := channel.New(
			store1.Aspen,
			gorp.Wrap(store1.Aspen),
			store1.Cesium,
			channelNet.RouteUnary(""),
		)
		channels, err := channelSvc.NewCreate().
			WithName("SG02").
			WithDataRate(25*telem.Hz).
			WithDataType(telem.Float64).
			WithNodeID(1).
			ExecN(ctx, 1)

		Expect(err).ToNot(HaveOccurred())
		factory = seg.NewSequentialFactory(dataFactory, 10*telem.Second, channels[0].Cesium)
		wrapper = &core.CesiumWrapper{KeyMap: map[cesium.ChannelKey]channel.
			Key{channels[0].Cesium.Key: channels[0].Key()}}

		keys := channel.Keys{channels[0].Key()}

		w, err = writer.New(
			ctx,
			store1.Cesium,
			channelSvc,
			store1.Aspen,
			net.RouteStream("", 0),
			keys,
		)
		ctx, _ := signal.Background()
		Expect(err).ToNot(HaveOccurred())
		req := confluence.NewStream[writer.Request](0)
		res := confluence.NewStream[writer.Response](0)
		w.OutTo(res)
		w.InFrom(req)
		w.Flow(ctx)
		requests = req
		responses = res
	})
	Context("Behavioral Accuracy", func() {
		It("Should write a segment to disk", func() {
			seg := factory.NextN(1)
			requests.AcquireInlet() <- writer.Request{Segments: wrapper.Wrap(seg)}
			requests.Close()
		})
	})
})
