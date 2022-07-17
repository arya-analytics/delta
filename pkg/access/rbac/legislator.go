package rbac

import (
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/x/gorp"
)

type Effect uint8

const (
	Deny Effect = iota
	Allow
)

type Legislator struct{ DB *gorp.DB }

func (l *Legislator) Create(txn gorp.Txn, p Policy) error {
	return gorp.NewCreate[string, Policy]().Entry(&p).Exec(txn)
}

func (l *Legislator) RetrieveBySubject(subject ontology.ID) (p []Policy, err error) {
	return p, gorp.NewRetrieve[string, Policy]().
		Where(func(p Policy) bool { return p.Subject == subject }).
		Entries(&p).
		Exec(l.DB)
}
