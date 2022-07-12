package sec

import (
	"github.com/arya-analytics/x/address"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
)

var (
	AccessDenied = errors.New("[sec] - access denied")
)

type AccessEnforcer interface {
	Enforce(subject Subject, resource Resource, action Action) error
}

type ResourceType uint16

type Resource struct {
	Key  string
	Type ResourceType
}

func RouteResource(path address.Address) {

}

type Action struct{}

type SubjectType uint16

type Subject struct {
	Key  string
	Type SubjectType
}

const (
	SubjectTypeUser SubjectType = iota
)

func NewUserSubject(key uuid.UUID) Subject {
	return Subject{Key: key.String(), Type: SubjectTypeUser}
}
