package rbac

import (
	"github.com/arya-analytics/delta/pkg/sec"
	"github.com/cockroachdb/errors"
)

type enforcer struct {
	store Storage
}

func NewEnforcer(storage Storage) sec.AccessEnforcer { return &enforcer{store: storage} }

func (e *enforcer) Enforce(subject, object, action sec.AccessEntity) error {
	ok, err := e.store.get(e.parsePolicy(subject, object, action))
	if err != nil || !ok {
		return errors.CombineErrors(sec.AccessDenied, err)
	}
	return nil
}

func (e *enforcer) parsePolicy(subject, action, object sec.AccessEntity) policy {
	var (
		subjKey   = e.getKey(subject)
		objKey    = e.getKey(object)
		actionKey = e.getKey(action)
	)
	return policy{subjKey, objKey, actionKey}
}

func (e *enforcer) getKey(entity sec.AccessEntity) string {
	key, ok := entity.GetAttr("key")
	if !ok {
		return ""
	}
	return key.(string)
}
