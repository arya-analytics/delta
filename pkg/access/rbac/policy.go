package rbac

import (
	"github.com/arya-analytics/delta/pkg/access"
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/x/filter"
)

type Policy struct {
	Subject ontology.ID
	Object  ontology.ID
	Actions []access.Action
	Effect  access.Effect
}

func NewPolicyKey(subject, object ontology.ID) string {
	return subject.String() + "-" + object.String()
}

// GorpKey implements the gorp.Entry interface.
func (p Policy) GorpKey() string { return NewPolicyKey(p.Subject, p.Object) }

// SetOptions implements the gorp.Entry interface.
func (p Policy) SetOptions() []interface{} { return nil }

// Matches returns true if the policy matches the given access.Request.
func (p Policy) Matches(req access.Request) bool {
	return p.Subject == req.Subject &&
		p.Object == req.Object &&
		filter.ElementOf(p.Actions, req.Action)
}
