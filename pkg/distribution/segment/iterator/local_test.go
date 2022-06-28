package iterator_test

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/cesium/testutil/seg"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/gorp"
	tmock "github.com/arya-analytics/x/transport/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"time"
)

var _ = Describe("Local", Ordered, func() {
	var (
		log        *zap.Logger
		builder    *mock.StorageBuilder
		channelSvc *channel.Service
		channels   []channel.Channel
		keys       channel.Keys
		net        *tmock.Network[iterator.Request, iterator.Response]
	)
	BeforeAll(func() {
		log = zap.NewNop()
		builder = mock.NewStorage()
		dataFactory := &seg.RandomFloat64Factory{Cache: true}

		net = tmock.NewNetwork[iterator.Request, iterator.Response]()

		channelNet := tmock.NewNetwork[channel.CreateMessage, channel.CreateMessage]()

		store1, err := builder.New(log)
		Expect(err).ToNot(HaveOccurred())

		channelSvc = channel.New(
			store1.Aspen,
			gorp.Wrap(store1.Aspen),
			store1.Cesium,
			channelNet.RouteUnary(""),
		)
		channels, err = channelSvc.NewCreate().
			WithName("SG02").
			WithDataRate(250*cesium.Hz).
			WithDataType(cesium.Float32).
			WithNodeID(1).
			ExecN(ctx, 10)

		Expect(err).ToNot(HaveOccurred())

		for _, ch := range channels {
			keys = append(keys, ch.Key())
			req, res, err := store1.Cesium.NewCreate().WhereChannels(ch.Key().Cesium()).Stream(ctx)
			Expect(err).ToNot(HaveOccurred())
			stc := &seg.StreamCreate{
				Req:               req,
				Res:               res,
				SequentialFactory: seg.NewSequentialFactory(dataFactory, 100*cesium.Second, ch.Cesium),
			}
			stc.CreateCRequestsOfN(10, 1)
			Expect(stc.CloseAndWait()).To(Succeed())
		}
		store1.Cesium.NewCreate().WhereChannels()
	})
	AfterAll(func() { Expect(builder.Close()).To(Succeed()) })
	It("Should correctly iterator through the segments data", func() {
		store := builder.Stores[1]
		iter, err := iterator.New(
			store.Cesium,
			channelSvc,
			store.Aspen,
			net.RouteStream("", 0),
			cesium.TimeRangeMax,
			keys,
		)
		Expect(err).To(BeNil())
		values := confluence.NewStream[iterator.Response](11)
		iter.OutTo(values)
		t0 := time.Now()
		var segments []core.Segment
		Expect(iter.First()).To(BeTrue())
		for i := 0; i < 10; i++ {
			res := <-values.Outlet()
			Expect(res.Segments).To(HaveLen(1))
			Expect(res.Error).To(BeNil())
			segments = append(segments, res.Segments...)
		}
		Expect(iter.Next()).To(BeTrue())
		for i := 0; i < 10; i++ {
			res := <-values.Outlet()
			Expect(res.Segments).To(HaveLen(1))
			Expect(res.Error).To(BeNil())
			segments = append(segments, res.Segments...)
		}
		logrus.Info("First: ", time.Since(t0))
	})
	FIt("Should exhaust the iterator", func() {
		store := builder.Stores[1]
		iter, err := iterator.New(
			store.Cesium,
			channelSvc,
			store.Aspen,
			net.RouteStream("", 0),
			cesium.TimeRangeMax,
			keys,
		)
		Expect(err).To(BeNil())
		values := confluence.NewStream[iterator.Response](11)
		iter.OutTo(values)
		t0 := time.Now()
		iter.First()
		iter.Exhaust()
		var segments []core.Segment
		count := 0
		for res := range values.Outlet() {
			count++
			Expect(res.Error).To(BeNil())
			segments = append(segments, res.Segments...)
			if count >= 100 {
				break
			}
		}
		logrus.Info(time.Since(t0))
		logrus.Info(len(segments[0].Segment.Data) * len(segments))
	})
})
