package sec

import "github.com/cockroachdb/errors"

type AccessEntity interface {
	GetAttr(key string) (interface{}, bool)
}

var (
	AccessDenied = errors.New("[sec] - access denied")
)

type AccessEnforcer interface {
	Enforce(subject, object, action AccessEntity) error
}
