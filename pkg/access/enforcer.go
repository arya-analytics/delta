package access

import "github.com/arya-analytics/delta/pkg/resource"

type Request struct {
	Subject resource.Key
	Object  resource.Key
	Action  Action
}

type Enforcer interface {
	Enforce(requests ...Request) error
}
