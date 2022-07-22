package storage_test

import (
	"github.com/arya-analytics/delta/pkg/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Storage", func() {
	Describe("Open", func() {
		Describe("Acquiring a lock", func() {
			It("Should return an error if the lock is already acquired", func() {
				cfg := storage.Config{
					Dirname:  "testdata/storage",
					KVEngine: storage.PebbleKV,
					TSEngine: storage.CesiumTS,
				}
				_, err := storage.Open(cfg)
				Expect(err).ToNot(HaveOccurred())
				_, err = storage.Open(cfg)
				Expect(err).To(HaveOccurred())

			})
		})
	})
})
