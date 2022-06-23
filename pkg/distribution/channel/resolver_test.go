package channel_test

import (
	"github.com/arya-analytics/aspen"
	aspenmock "github.com/arya-analytics/aspen/mock"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/gorp"
	tmock "github.com/arya-analytics/x/transport/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Resolver", func() {
	var (
		resolver channel.Resolver
		key      channel.Key
	)
	BeforeEach(func() {
		log := zap.NewNop()
		net := tmock.NewNetwork[channel.CreateMessage, channel.CreateMessage]()
		aspenBuilder := aspenmock.NewMemBuilder(aspen.WithLogger(log.Sugar()))
		db1, err := aspenBuilder.New()
		Expect(err).ToNot(HaveOccurred())
		cdb1, err := cesium.Open("", cesium.MemBacked(), cesium.WithLogger(log))
		Expect(err).ToNot(HaveOccurred())
		ch, err := channel.New(db1, gorp.Wrap(db1), cdb1, net.RouteUnary("")).
			NewCreate().
			WithDataRate(5 * cesium.Hz).
			WithDataType(cesium.Float64).
			WithName("SG01").
			WithNodeID(1).
			Exec(ctx)
		Expect(err).ToNot(HaveOccurred())
		key = ch.Key()

		db2, err := aspenBuilder.New()
		Expect(err).ToNot(HaveOccurred())
		cdb2, err := cesium.Open("", cesium.MemBacked(), cesium.WithLogger(log))
		Expect(err).ToNot(HaveOccurred())
		resolver = channel.New(db2, gorp.Wrap(db2), cdb2, net.RouteUnary(""))
	})
	It("Should correctly resolve the address of the channel", func() {
		addr, err := resolver.Resolve(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(addr).To(Equal(address.Address("localhost:0")))
	})
})
