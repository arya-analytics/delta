package writer_test

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/cesium/testutil/seg"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/lock"
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
		factory   seg.SequentialFactory
		wrapper   *core.CesiumWrapper
		keys      channel.Keys
		newWriter func() (writer.Writer, error)
	)
	BeforeAll(func() {
		log = zap.NewNop()
		builder = mock.NewStorage()
		dataFactory := &seg.RandomFloat64Factory{Cache: true}

		net = tmock.NewNetwork[writer.Request, writer.Response]()

		channelNet := tmock.NewNetwork[channel.CreateRequest, channel.CreateRequest]()

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

		keys = channel.Keys{channels[0].Key()}

		newWriter = func() (writer.Writer, error) {
			return writer.New(
				ctx,
				store1.Cesium,
				channelSvc,
				store1.Aspen,
				net.RouteStream("", 0),
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
		It("Should write a segment to disk", func() {
			seg := factory.NextN(1)
			w.Requests() <- writer.Request{Segments: wrapper.Wrap(seg)}
			close(w.Requests())
			for res := range w.Responses() {
				Expect(res.Error).ToNot(HaveOccurred())
			}
			Expect(w.Close()).To(Succeed())
		})
		It("Should write multiple segments to disk", func() {
			seg := factory.NextN(10)
			w.Requests() <- writer.Request{Segments: wrapper.Wrap(seg)}
			close(w.Requests())
			for res := range w.Responses() {
				Expect(res.Error).ToNot(HaveOccurred())
			}
			Expect(w.Close()).To(Succeed())
		})
		It("Should return an error when another writer has a lock on the channel", func() {
			_, err := newWriter()
			Expect(err).To(MatchError(lock.ErrLocked))
		})
	})
})
