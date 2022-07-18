package ontology_test

import (
	"github.com/arya-analytics/delta/pkg/ontology"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Retrieve", func() {
	var w ontology.Writer
	BeforeEach(func() { w = otg.NewWriter(txn) })
	Describe("Single Clause", func() {
		It("Should retrieve a resource by its ID", func() {
			id := ontology.ID{Key: "foo", Type: "bar"}
			Expect(w.DefineResource(id)).To(Succeed())
		})
	})
})
