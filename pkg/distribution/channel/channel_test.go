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
		It("Should return the correct channel key", func() {
			c := channel.Channel{
				NodeID: 1,
				Cesium: cesium.Channel{
					Key: 2,
				},
			}
			key := c.Key()
			Expect(key.NodeID()).To(Equal(aspen.NodeID(1)))
			Expect(key.ChannelKey()).To(Equal(cesium.ChannelKey(2)))
		})
	})
})
