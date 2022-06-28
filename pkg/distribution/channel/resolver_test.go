package channel_test

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/mock"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/gorp"
	tmock "github.com/arya-analytics/x/transport/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Resolver", Ordered, func() {
	var (
		resolver channel.Resolver
		key      channel.Key
		builder  *mock.StorageBuilder
	)
	BeforeAll(func() {
		log := zap.NewNop()
		net := tmock.NewNetwork[channel.CreateMessage, channel.CreateMessage]()
		builder = mock.NewStorage()
		store1, err := builder.New(log)
		Expect(err).To(BeNil())
		Expect(err).ToNot(HaveOccurred())
		ch, err := channel.New(store1.Aspen, gorp.Wrap(store1.Aspen), store1.Cesium, net.RouteUnary("")).
			NewCreate().
			WithDataRate(5 * cesium.Hz).
			WithDataType(cesium.Float64).
			WithName("SG01").
			WithNodeID(1).
			Exec(ctx)
		Expect(err).ToNot(HaveOccurred())
		key = ch.Key()
		store2, err := builder.New(log)
		Expect(err).To(BeNil())
		resolver = channel.New(store2.Aspen, gorp.Wrap(store2.Aspen), store2.Cesium, net.RouteUnary(""))
	})
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
	})
	It("Should correctly resolve the address of the channel", func() {
		addr, err := resolver.Resolve(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(addr).To(Equal(address.Address("localhost:0")))
	})
})
