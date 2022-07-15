package node

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/delta/pkg/resource"
	"strconv"
)

type (
	Node = aspen.Node
	ID   = aspen.NodeID
)

func ResourceKey(id ID) resource.Key {
	return resource.Key{Type: ResourceType, Key: strconv.Itoa(int(id))}
}
