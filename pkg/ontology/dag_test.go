package ontology_test

import (
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv/memkv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dag", Ordered, func() {
	var dag *ontology.DAG
	BeforeAll(func() {
		dag = &ontology.DAG{DB: gorp.Wrap(memkv.Open()).BeginTxn()}
	})
	It("Should prevent circular relationships", func() {
		aKey := ontology.Key{Key: "a", Type: "a"}
		bKey := ontology.Key{Key: "b", Type: "b"}
		cKey := ontology.Key{Key: "c", Type: "c"}
		Expect(dag.DefineResource(aKey)).To(Succeed())
		Expect(dag.DefineResource(bKey)).To(Succeed())
		Expect(dag.DefineResource(cKey)).To(Succeed())
		Expect(dag.DefineRelationship(aKey, bKey)).To(Succeed())
		Expect(dag.DefineRelationship(bKey, cKey)).To(Succeed())
		err := dag.DefineRelationship(cKey, aKey)
		Expect(err).To(HaveOccurred())
	})
})
