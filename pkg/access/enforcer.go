package access

import "github.com/arya-analytics/delta/pkg/ontology"

type Request struct {
	Subject ontology.Key
	Object  ontology.Key
	Action  Action
}

type Enforcer interface {
	Enforce(requests ...Request) error
}
