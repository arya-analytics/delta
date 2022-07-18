package ontology_test

import (
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv/memkv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	db  *gorp.DB
	otg *ontology.Ontology
	txn gorp.Txn
)

var _ = BeforeSuite(func() {
	var err error
	db = gorp.Wrap(memkv.New())
	otg, err = ontology.Open(gorp.Wrap(db))
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(db.Close()).To(Succeed())
})

var _ = BeforeEach(func() {
	txn = db.BeginTxn()
})

var _ = AfterEach(func() {
	Expect(txn.Close()).To(Succeed())
})

func TestOntology(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ontology Suite")
}
