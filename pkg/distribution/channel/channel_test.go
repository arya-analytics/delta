package channel_test

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/x/binary"
	"github.com/arya-analytics/x/telem"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Channel", func() {
	Describe("Name", func() {
		var key channel.Key
		BeforeEach(func() {
			key = channel.NewKey(1, 2)
		})
		Describe("NodeID", func() {
			It("Should return the correct node ID for the channel", func() {
				Expect(key.NodeID()).To(Equal(aspen.NodeID(1)))
			})
		})
		Describe("Cesium", func() {
			It("Should return the correct cesium key for the channel", func() {
				Expect(key.Cesium()).To(Equal(cesium.ChannelKey(2)))
			})
		})
	})
	Describe("Encoding + Decoding", func() {
		It("Should encode and decode a channel correctly", func() {
			ch := channel.Channel{
				NodeID: 1,
				Cesium: cesium.Channel{
					Key:      2,
					DataRate: 5 * telem.Hz,
					DataType: telem.Float32,
				},
			}
			ed := &binary.GobEncoderDecoder{}
			encoded, err := ed.Encode(ch)
			Expect(err).To(BeNil())
			var decoded channel.Channel
			Expect(ed.Decode(encoded, &decoded)).To(BeNil())
			Expect(decoded).To(Equal(ch))
		})
	})
})
