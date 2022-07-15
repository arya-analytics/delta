package resource_test

import (
	"github.com/arya-analytics/delta/pkg/resource"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv/memkv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dag", Ordered, func() {
	var dag *resource.DAG
	BeforeAll(func() {
		dag = &resource.DAG{Txn: gorp.Wrap(memkv.Open()).BeginTxn()}
	})
	It("Should prevent circular relationships", func() {
		aKey := resource.Key{Key: "a", Type: "a"}
		bKey := resource.Key{Key: "b", Type: "b"}
		cKey := resource.Key{Key: "c", Type: "c"}
		Expect(dag.DefineResource(aKey)).To(Succeed())
		Expect(dag.DefineResource(bKey)).To(Succeed())
		Expect(dag.DefineResource(cKey)).To(Succeed())
		Expect(dag.DefineRelationship(aKey, bKey)).To(Succeed())
		Expect(dag.DefineRelationship(bKey, cKey)).To(Succeed())
		err := dag.DefineRelationship(cKey, aKey)
		Expect(err).To(HaveOccurred())
	})
})
