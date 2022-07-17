package ontology_test

import (
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv/pebblekv"
	"github.com/cockroachdb/pebble"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"time"
)

var _ = FDescribe("Retrieve", func() {
	var otg *ontology.Ontology
	BeforeEach(func() {
		var err error
		pdb, err := pebble.Open("./testdata", &pebble.Options{})
		otg, err = ontology.Open(gorp.Wrap(pebblekv.Wrap(pdb)))
		Expect(err).ToNot(HaveOccurred())
	})
	AfterEach(func() {
		Expect(otg.DB.Close()).To(Succeed())
	})
	It("Should retrieve a resource by its ID", func() {
		id := ontology.ID{Key: "a", Type: "a"}
		Expect(otg.NewWriter(otg.DB).DefineResource(id)).To(Succeed())
		res := &ontology.Resource{}
		Expect(otg.NewRetrieve().Entry(res).WhereIDs(id).Exec()).To(Succeed())
	})
	It("Should retrieve the parents of a resource", func() {
		var (
			childID  = ontology.ID{Key: "a", Type: "a"}
			parentID = ontology.ID{Key: "b", Type: "b"}
		)
		Expect(otg.NewWriter(otg.DB).DefineResource(childID)).To(Succeed())
		Expect(otg.NewWriter(otg.DB).DefineResource(parentID)).To(Succeed())
		Expect(otg.NewWriter(otg.DB).DefineRelationship(
			childID,
			parentID,
			ontology.Parent,
		)).To(Succeed())
		var parents []ontology.Resource
		err := otg.NewRetrieve().
			Entry(&ontology.Resource{}).
			WhereIDs(childID).
			TraverseTo(ontology.Parents).
			Entries(&parents).
			Exec()
		Expect(err).ToNot(HaveOccurred())
	})
	FIt("Should retrieve the grandparents of a resource", func() {
		var (
			grandID  = ontology.ID{Key: "c", Type: "c"}
			parentID = ontology.ID{Key: "b", Type: "b"}
			childID  = ontology.ID{Key: "a", Type: "a"}
		)
		Expect(otg.NewWriter(otg.DB).DefineResource(childID)).To(Succeed())
		Expect(otg.NewWriter(otg.DB).DefineResource(parentID)).To(Succeed())
		Expect(otg.NewWriter(otg.DB).DefineResource(grandID)).To(Succeed())
		Expect(otg.NewWriter(otg.DB).DefineRelationship(
			childID,
			parentID,
			ontology.Parent,
		)).To(Succeed())
		Expect(otg.NewWriter(otg.DB).DefineRelationship(
			parentID,
			grandID,
			ontology.Parent,
		)).To(Succeed())
		var grandparents []ontology.Resource
		t0 := time.Now()
		err := otg.NewRetrieve().
			Entry(&ontology.Resource{}).
			WhereIDs(childID).
			TraverseTo(ontology.Parents).
			Entry(&ontology.Resource{}).
			TraverseTo(ontology.Parents).
			Entries(&grandparents).
			Exec()
		logrus.Info(time.Since(t0))
		Expect(err).ToNot(HaveOccurred())
	})
})
