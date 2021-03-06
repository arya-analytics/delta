package iterator_test

import (
	"github.com/arya-analytics/cesium/testutil/seg"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/telem"
	tmock "github.com/arya-analytics/x/transport/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Local", Ordered, func() {
	var (
		log     *zap.Logger
		net     *tmock.Network[iterator.Request, iterator.Response]
		iter    iterator.Iterator
		builder *mock.StorageBuilder
	)
	BeforeAll(func() {
		log = zap.NewNop()
		builder = mock.NewStorage()
		dataFactory := &seg.RandomFloat64Factory{Cache: true}

		net = tmock.NewNetwork[iterator.Request, iterator.Response]()

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

		var keys channel.Keys
		for _, ch := range channels {
			keys = append(keys, ch.Key())
			req, res, err := store1.Cesium.NewCreate().WhereChannels(ch.Key().Cesium()).Stream(ctx)
			Expect(err).ToNot(HaveOccurred())
			stc := &seg.StreamCreate{
				Req:               req,
				Res:               res,
				SequentialFactory: seg.NewSequentialFactory(dataFactory, 10*telem.Second, ch.Cesium),
			}
			stc.CreateCRequestsOfN(10, 1)
			Expect(stc.CloseAndWait()).To(Succeed())
		}

		iter, err = iterator.New(
			ctx,
			store1.Cesium,
			channelSvc,
			store1.Aspen,
			net.RouteStream("", 0),
			telem.TimeRangeMax,
			keys,
		)
		Expect(err).ToNot(HaveOccurred())
	})
	AfterAll(func() {
		Expect(iter.Close()).To(Succeed())
		_, ok := <-iter.Responses()
		Expect(ok).To(BeFalse())
		Expect(builder.Close()).To(Succeed())

	})
	// Behavioral accuracy tests check whether the iterator returns the correct
	// boolean acknowledgements and segment counts. These tests DO NOT check
	// for data accuracy.
	Context("Behavioral Accuracy", func() {
		Describe("First", func() {
			It("Should return the first segment in the iterator", func() {
				Expect(iter.First()).To(BeTrue())
				res := <-iter.Responses()
				Expect(res.Error).To(BeNil())
				Expect(res.Segments).To(HaveLen(1))
			})
		})
		Describe("SeekFirst + TraverseTo", func() {
			It("Should return the next segment in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next()).To(BeTrue())
				res := <-iter.Responses()
				Expect(res.Error).To(BeNil())
				Expect(res.Segments).To(HaveLen(1))
			})
		})
		Describe("SeekLast + Prev", func() {
			It("Should return the previous segment in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Prev()).To(BeTrue())
				res := <-iter.Responses()
				Expect(res.Error).To(BeNil())
				Expect(res.Segments).To(HaveLen(1))
			})
		})
		Describe("NextSpan", func() {
			It("Should return the next span in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.NextSpan(20 * telem.Second)).To(BeTrue())
				res := <-iter.Responses()
				Expect(res.Error).To(BeNil())
				Expect(res.Segments).To(HaveLen(1))
				res2 := <-iter.Responses()
				Expect(res2.Error).To(BeNil())
				Expect(res2.Segments).To(HaveLen(1))
			})
		})
		Describe("PrevSpan", func() {
			It("Should return the previous span in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.PrevSpan(20 * telem.Second)).To(BeTrue())
				res := <-iter.Responses()
				Expect(res.Error).To(BeNil())
				Expect(res.Segments).To(HaveLen(1))
				res2 := <-iter.Responses()
				Expect(res2.Error).To(BeNil())
				Expect(res2.Segments).To(HaveLen(1))
			})
		})
	})
})
