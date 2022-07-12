package access

import (
	"github.com/arya-analytics/x/path"
	"github.com/arya-analytics/x/set"
)

// Resource represents an entity that can be accessed. A resource is identified by a path
// within a set of parent resources. For example, a channel connected to a strain gauge
// would be identified by the path "/channel/sg/01". All resources are ultimately children
// of the root resource "/". ResourcePaths can be identified by several paths, but
// all paths must:
//
//		1. Be unique to the resource i.e. the path is f(path) -> resource is injective.
//		2. Terminate in the same key. This is enforced by the resource struct,
//		where the Key is always the last element in the path.
//
type Resource struct {
	// Key is the last element in the resource path, and should be the most granular
	// element in the path. A Key must be unique to all resources in the same
	// parent Resource.
	Key string
	// Parents is a set of parent resources that the Resource is a child of.
	Parents set.Set[path.Path]
	// Attributes is a map of key-value attributes that can be used to identify the
	// resource.
	Attributes map[string]interface{}
}

// RootResource is the parent of all resources.
var RootResource = Resource{Key: "/"}
