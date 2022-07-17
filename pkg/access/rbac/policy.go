package rbac

import (
	"github.com/arya-analytics/delta/pkg/access"
	"github.com/arya-analytics/delta/pkg/ontology"
)

type Policy struct {
	Subject ontology.ID
	Object  ontology.ID
	Actions []access.Action
	Effect  Effect
}

func NewPolicyKey(subject, object ontology.ID) string {
	return subject.String() + "-" + object.String()
}

// GorpKey implements the gorp.Entry interface.
func (p Policy) GorpKey() string { return NewPolicyKey(p.Subject, p.Object) }

// SetOptions implements the gorp.Entry interface.
func (p Policy) SetOptions() []interface{} { return nil }
