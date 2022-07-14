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

func ResourceTypeKey(id ID) resource.TypeKey {
	return resource.TypeKey{Type: ResourceType, Key: strconv.Itoa(int(id))}
}
