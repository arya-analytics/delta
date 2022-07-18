package access

import "github.com/arya-analytics/delta/pkg/ontology"

type Request struct {
	Subject ontology.ID
	Object  ontology.ID
	Action  Action
}

type Enforcer interface {
	Enforce(req Request) error
}
