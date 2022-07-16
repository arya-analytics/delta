package rbac

import (
	"github.com/arya-analytics/delta/pkg/access"
	"github.com/arya-analytics/x/query"
)

type enforcer struct {
	def Effect
	leg *Legislator
}

func (e *enforcer) Enforce(req access.Request) error {
	policy, err := e.leg.RetrieveBySubject(NewPolicyKey(req.Subject, req.Object))
	if err == query.NotFound {
		if e.def == Deny {
			return access.Forbidden
		}
		return nil
	}
	if err != nil {
		return err
	}
}
