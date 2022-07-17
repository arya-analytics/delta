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
		aKey := ontology.ID{Key: "a", Type: "a"}
		bKey := ontology.ID{Key: "b", Type: "b"}
		cKey := ontology.ID{Key: "c", Type: "c"}
		Expect(dag.DefineResource(aKey)).To(Succeed())
		Expect(dag.DefineResource(bKey)).To(Succeed())
		Expect(dag.DefineResource(cKey)).To(Succeed())
		Expect(dag.DefineRelationship(aKey, bKey, ontology.Parent)).To(Succeed())
		Expect(dag.DefineRelationship(bKey, cKey, ontology.Parent)).To(Succeed())
		err := dag.DefineRelationship(cKey, aKey, ontology.Parent)
		Expect(err).To(HaveOccurred())
	})
})
