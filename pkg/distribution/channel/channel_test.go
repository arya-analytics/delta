package channel_test

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Channel", func() {
	Describe("Key", func() {
		var key channel.Key
		BeforeEach(func() {
			key = channel.NewKey(1, 2)
		})
		Describe("NodeID", func() {
			It("Should return the correct node ID for the channel", func() {
				Expect(key.NodeID()).To(Equal(aspen.NodeID(1)))
			})
		})
		Describe("CesiumKey", func() {
			It("Should return the correct cesium key for the channel", func() {
				Expect(key.CesiumKey()).To(Equal(cesium.ChannelKey(2)))
			})
		})
	})
})
