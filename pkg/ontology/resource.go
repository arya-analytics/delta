package ontology

import (
	"fmt"
	"github.com/arya-analytics/delta/pkg/ontology/schema"
)

// ID is a unique identifier for a Resource. An example:
//
// userKey := ID{
//     ID:  "748d31e2-5732-4cb5-8bc9-64d4ad51efe8",
//     Type: "user",
// }
//
// They key has two elements so for two reasons. First, by storing the Type we know which
// Service to query for additional info on the Resource. Second, while a ID.Key may be
// unique for a particular resource (e.g. channel), it might not be unique across ALL
// resources. We need something universally unique across the entire delta cluster.
type ID struct {
	// Key is a string that uniquely identifies a Resource within its Type.
	Key string
	// Type defines the type of Resource the Key refers to :). For example,
	// a channel is a Resource of type "channel". A user is a Resource of type
	// "user".
	Type Type
}

func (k ID) Validate() error {
	if k.Key == "" {
		return fmt.Errorf("[resource] - key is required")
	}
	if k.Type == "" {
		return fmt.Errorf("[resource] - type is required")
	}
	return nil
}

func (k ID) String() string {
	return fmt.Sprintf("%s:%s", k.Key, k.Type)
}

type Resource struct {
	Key  ID
	data schema.Entity
}

// GorpKey implements the gorp.Entry interface.
func (r Resource) GorpKey() ID { return r.Key }

// SetOptions implements the gorp.Entry interface.
func (r Resource) SetOptions() []interface{} { return nil }
