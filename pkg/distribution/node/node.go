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

func ResourceKey(id ID) ontology.Key {
	return ontology.Key{Type: ResourceType, Key: strconv.Itoa(int(id))}
}
