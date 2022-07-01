package iterator_test

import (
	"github.com/arya-analytics/cesium/testutil/seg"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
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
		log     *zap.Logger
		net     *tmock.Network[iterator.Request, iterator.Response]
		iter    iterator.Iterator
		builder *mock.StorageBuilder
		values  confluence.Outlet[iterator.Response]
		nRes    int
	)
	BeforeAll(func() {
		log = zap.NewNop()
		builder = mock.NewStorage()
		dataFactory := &seg.RandomFloat64Factory{Cache: true}
		net = tmock.NewNetwork[iterator.Request, iterator.Response]()
		channelNet := tmock.NewNetwork[channel.CreateMessage, channel.CreateMessage]()

		node1Addr := address.Address("localhost:0")
		node2Addr := address.Address("localhost:1")

		store1, err := builder.New(log)
		Expect(err).ToNot(HaveOccurred())
		node1Transport := net.RouteStream(node1Addr, 0)
		iterator.NewServer(store1.Cesium, store1.Aspen.HostID(), node1Transport)

		store2, err := builder.New(log)
		Expect(err).ToNot(HaveOccurred())
		node2Transport := net.RouteStream(node2Addr, 0)
		iterator.NewServer(store2.Cesium, store2.Aspen.HostID(), node2Transport)

		channelSvc := channel.New(
			store1.Aspen,
			gorp.Wrap(store1.Aspen),
			store1.Cesium,
			channelNet.RouteUnary(node1Addr),
		)
		dr := 25 * telem.Hz
		channels, err := channelSvc.NewCreate().
			WithName("SG02").
			WithDataRate(dr).
			WithDataType(telem.Float64).
			WithNodeID(1).
			ExecN(ctx, 1)

		var keys channel.Keys
		dur := 10 * telem.Second
		nReq := 10
		nSeg := 10
		nRes = nReq * nSeg
		for _, ch := range channels {
			keys = append(keys, ch.Key())
			req, res, err := store1.Cesium.NewCreate().WhereChannels(ch.Key().Cesium()).Stream(ctx)
			Expect(err).ToNot(HaveOccurred())
			stc := &seg.StreamCreate{
				Req: req,
				Res: res,
				SequentialFactory: seg.NewSequentialFactory(dataFactory, dur,
					ch.Cesium),
			}
			stc.CreateCRequestsOfN(nReq, nSeg)
			Expect(stc.CloseAndWait()).ToNot(HaveOccurred())
		}

		time.Sleep(50 * time.Millisecond)

		store2ChannelSvc := channel.New(
			store2.Aspen,
			gorp.Wrap(store2.Aspen),
			store2.Cesium,
			channelNet.RouteUnary(node2Addr),
		)

		iter, err = iterator.New(
			store2.Cesium,
			store2ChannelSvc,
			store2.Aspen,
			node2Transport,
			telem.TimeRangeMax,
			keys,
		)
		Expect(err).ToNot(HaveOccurred())
		v := confluence.NewStream[iterator.Response](10)
		iter.OutTo(v)
		values = v
	})
	AfterAll(func() {
		_, ok := <-values.Outlet()
		Expect(ok).To(BeFalse())
		Expect(builder.Close()).To(Succeed())
	})
	Context("Behavioral Accuracy", func() {
		Describe("First", func() {
			It("Should return the first segment in the iterator", func() {
				Expect(iter.First()).To(BeTrue())
				res := <-values.Outlet()
				Expect(res.Segments).To(HaveLen(1))
			})
		})
		Describe("SeekFirst + Next", func() {
			It("Should return the first segment in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next()).To(BeTrue())
				res := <-values.Outlet()
				Expect(res.Segments).To(HaveLen(1))
			})
		})
		Describe("SeekLast + Prev", func() {
			It("Should return the last segment in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Prev()).To(BeTrue())
				res := <-values.Outlet()
				Expect(res.Segments).To(HaveLen(1))
			})
		})
		Describe("NextSpan", func() {
			It("Should return the next span in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.NextSpan(20 * telem.Second)).To(BeTrue())
				res := <-values.Outlet()
				Expect(res.Segments).To(HaveLen(1))
				res2 := <-values.Outlet()
				Expect(res2.Segments).To(HaveLen(1))
			})
		})
		Describe("PrevSpan", func() {
			It("Should return the previous span in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.PrevSpan(20 * telem.Second)).To(BeTrue())
				res := <-values.Outlet()
				Expect(res.Segments).To(HaveLen(1))
				res2 := <-values.Outlet()
				Expect(res2.Segments).To(HaveLen(1))
			})
		})
		Describe("NextRange", func() {
			It("Should return the next range of data in the iterator", func() {
				Expect(iter.NextRange(telem.TimeRange{
					Start: 0,
					End:   telem.TimeStamp(30 * telem.Second),
				})).To(BeTrue())
				res1 := <-values.Outlet()
				Expect(res1.Segments).To(HaveLen(1))
				res2 := <-values.Outlet()
				Expect(res2.Segments).To(HaveLen(1))
				res3 := <-values.Outlet()
				Expect(res3.Segments).To(HaveLen(1))
			})
		})
		Describe("Exhausting the Iterator", func() {
			It("Should return ten segments", func() {
				go func() {
					iter.Exhaust()
					Expect(iter.Close()).To(Succeed())
				}()
				segments := make([]core.Segment, 0, nRes)
				for res := range values.Outlet() {
					segments = append(segments, res.Segments...)
				}
				Expect(len(segments)).To(Equal(nRes))
			})
		})
	})
})
