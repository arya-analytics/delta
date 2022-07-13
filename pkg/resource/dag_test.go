package resource_test

import (
	"github.com/arya-analytics/delta/pkg/resource"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv/memkv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"time"
)

var _ = Describe("Dag", Ordered, func() {
	var dag *resource.DAG
	BeforeAll(func() {
		dag = resource.OpenDAG(gorp.Wrap(memkv.Open()))
	})
	It("Should construct a graph", func() {
		t0 := time.Now()
		Expect(dag.SetResource(resource.Resource{Key: "a", Type: "a"})).To(Succeed())
		Expect(dag.SetResource(resource.Resource{Key: "b", Type: "b"})).To(Succeed())
		Expect(dag.SetResource(resource.Resource{Key: "c", Type: "c"})).To(Succeed())
		Expect(dag.SetRelationship(resource.Relationship{Parent: "a", Child: "b"})).To(Succeed())
		Expect(dag.SetRelationship(resource.Relationship{Parent: "b", Child: "c"})).To(Succeed())
		err := dag.SetRelationship(resource.Relationship{Parent: "c", Child: "a"})
		logrus.Info(time.Since(t0))
		Expect(err).To(HaveOccurred())
	})
})
