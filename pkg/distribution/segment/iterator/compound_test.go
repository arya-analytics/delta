package iterator_test

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/cesium/testutil/seg"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/telem"
	tmock "github.com/arya-analytics/x/transport/mock"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"time"
)

func assertResponse(
	c,
	n int,
	iter iterator.Iterator,
	timeout time.Duration,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for i := 0; i < c; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case v := <-iter.Responses():
			if len(v.Segments) != n {
				return errors.Newf("expected %v segments, received %v", n, len(v.Segments))
			}
		}
	}
	select {
	case <-iter.Responses():
		return errors.Newf("expected no more iter, received extra response")
	case <-ctx.Done():
		return nil
	}
}

var _ = Describe("Compound", Ordered, func() {
	var (
		log       *zap.Logger
		net       *tmock.Network[iterator.Request, iterator.Response]
		iter      iterator.Iterator
		builder   *mock.StorageBuilder
		nChannels int
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
		var channels []channel.Channel
		node1Channels, err := channelSvc.NewCreate().
			WithName("SG02").
			WithDataRate(dr).
			WithDataType(telem.Float64).
			WithNodeID(1).
			ExecN(ctx, 1)
		Expect(err).ToNot(HaveOccurred())
		channels = append(channels, node1Channels...)

		store2ChannelSvc := channel.New(
			store2.Aspen,
			gorp.Wrap(store2.Aspen),
			store2.Cesium,
			channelNet.RouteUnary(node2Addr),
		)

		node2Channels, err := channelSvc.NewCreate().
			WithName("SG02").
			WithDataRate(dr).
			WithDataType(telem.Float64).
			WithNodeID(2).
			ExecN(ctx, 1)
		Expect(err).ToNot(HaveOccurred())
		channels = append(channels, node2Channels...)
		nChannels = len(channels)

		var keys channel.Keys
		dur := 10 * telem.Second
		nReq := 10
		nSeg := 10
		//nRes = nReq * nSeg * len(channels)
		for _, ch := range channels {
			var db cesium.DB
			if ch.NodeID == 1 {
				db = store1.Cesium
			} else {
				db = store2.Cesium
			}
			keys = append(keys, ch.Key())
			req, res := make(chan cesium.CreateRequest), make(chan cesium.CreateResponse)
			go func() {
				err := db.NewCreate().WhereChannels(ch.Key().Cesium()).
					Stream(ctx, req, res)
				Expect(err).ToNot(HaveOccurred())
			}()
			stc := &seg.StreamCreate{
				Req: req,
				Res: res,
				SequentialFactory: seg.NewSequentialFactory(dataFactory, dur,
					ch.Cesium),
			}
			stc.CreateCRequestsOfN(nReq, nSeg)
			Expect(stc.CloseAndWait()).ToNot(HaveOccurred())
		}

		time.Sleep(100 * time.Millisecond)

		iter, err = iterator.New(
			ctx,
			store2.Cesium,
			store2ChannelSvc,
			store2.Aspen,
			node2Transport,
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
	Context("Behavioral Accuracy", func() {
		Describe("First", func() {
			It("Should return the first segment in the iterator", func() {
				Expect(iter.First()).To(BeTrue())
				Expect(assertResponse(
					nChannels,
					1,
					iter,
					20*time.Millisecond,
				)).To(Succeed())
			})
		})
		Describe("SeekFirst + Next", func() {
			It("Should return the first segment in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next()).To(BeTrue())
				Expect(assertResponse(
					nChannels,
					1,
					iter,
					20*time.Millisecond,
				)).To(Succeed())
			})
		})
		Describe("SeekLast + Prev", func() {
			It("Should return the last segment in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Prev()).To(BeTrue())
				Expect(assertResponse(
					nChannels,
					1,
					iter,
					20*time.Millisecond,
				)).To(Succeed())
			})
		})
		Describe("NextSpan", func() {
			It("Should return the next span in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.NextSpan(20 * telem.Second)).To(BeTrue())
				Expect(assertResponse(
					nChannels*2,
					1,
					iter,
					20*time.Millisecond,
				)).To(Succeed())
			})
		})
		Describe("PrevSpan", func() {
			It("Should return the previous span in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.PrevSpan(20 * telem.Second)).To(BeTrue())
				Expect(assertResponse(
					nChannels*2,
					1,
					iter,
					20*time.Millisecond,
				)).To(Succeed())
			})
		})
		Describe("NextRange", func() {
			It("Should return the next range of data in the iterator", func() {
				Expect(iter.NextRange(telem.TimeRange{
					Start: 0,
					End:   telem.TimeStamp(30 * telem.Second),
				})).To(BeTrue())
				Expect(assertResponse(
					nChannels*3,
					1,
					iter,
					30*time.Millisecond,
				))
			})
		})
	})
})
