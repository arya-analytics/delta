package rbac

import (
	"github.com/arya-analytics/delta/pkg/access"
)

type enforcer struct{}

func (e *enforcer) Enforce(requests ...access.Request) error {
	for _, req := range requests {

	}
}

func (e *enforcer) enforceOne(req access.Request) error {
}
