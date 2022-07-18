package ontology_test

import (
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/delta/pkg/ontology/schema"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Retrieve", func() {
	var w ontology.Writer
	BeforeEach(func() { w = otg.NewWriter(txn) })
	Describe("Single Clause", func() {
		It("Should retrieve a resource by its ID", func() {
			id := newEmptyID("A")
			Expect(w.DefineResource(id)).To(Succeed())
			var r ontology.Resource
			Expect(w.NewRetrieve().
				WhereIDs(id).
				Entry(&r).
				Exec(),
			).To(Succeed())
			v, ok := schema.Get[string](r.Entity(), "key")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("A"))
		})
	})
})
