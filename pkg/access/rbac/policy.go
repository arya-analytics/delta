package rbac

import (
	"github.com/arya-analytics/delta/pkg/access"
	"github.com/arya-analytics/delta/pkg/ontology"
)

type Policy struct {
	Subject ontology.Key
	Object  ontology.Key
	Actions []access.Action
	Effect  Effect
}

func NewPolicyKey(subject, object ontology.Key) string {
	return subject.String() + "-" + object.String()
}

// GorpKey implements the gorp.Entry interface.
func (p Policy) GorpKey() string { return NewPolicyKey(p.Subject, p.Object) }

// SetOptions implements the gorp.Entry interface.
func (p Policy) SetOptions() []interface{} { return nil }
