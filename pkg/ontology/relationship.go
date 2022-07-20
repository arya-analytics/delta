package ontology

import (
	"fmt"
	"github.com/cockroachdb/errors"
)

type RelationshipType string

const (
	Parent RelationshipType = "parent"
)

type Relationship struct {
	From ID
	To   ID
	Type RelationshipType
}

func (r Relationship) GorpKey() string {
	return fmt.Sprintf("%s:%s:%s", r.From.String(), r.To.String(), r.Type)

}
func (r Relationship) SetOptions() []interface{} { return nil }

func (r Relationship) Validate() error {
	if r.From.Key == "" {
		return errors.Newf("[resource] - relationship from is required")
	}
	if r.To.Key == "" {
		return errors.Newf("[resource] - relationship to is required")
	}
	if r.From == r.To {
		return errors.Newf("[resource] - relationship to and from cannot be the same")
	}
	return nil
}
