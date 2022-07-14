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
		aKey := resource.TypeKey{Key: "a", Type: "a"}
		bKey := resource.TypeKey{Key: "b", Type: "b"}
		cKey := resource.TypeKey{Key: "c", Type: "c"}
		Expect(dag.SetResource(aKey)).To(Succeed())
		Expect(dag.SetResource(bKey)).To(Succeed())
		Expect(dag.SetResource(cKey)).To(Succeed())
		Expect(dag.SetRelationship(aKey, bKey)).To(Succeed())
		Expect(dag.SetRelationship(bKey, cKey)).To(Succeed())
		err := dag.SetRelationship(cKey, aKey)
		Expect(err).To(HaveOccurred())
	})
})
