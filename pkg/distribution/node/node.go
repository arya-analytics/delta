package node

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/delta/pkg/ontology"
	"strconv"
)

type (
	Node = aspen.Node
	ID   = aspen.NodeID
)

func OntologyID(id ID) ontology.ID {
	return ontology.ID{Type: ResourceType, Key: strconv.Itoa(int(id))}
}
