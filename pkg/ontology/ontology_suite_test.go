package ontology_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOntology(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ontology Suite")
}
