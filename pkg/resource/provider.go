package resource

import "github.com/arya-analytics/x/gorp"

type Provider interface {
	GetAttributes(txn gorp.Txn, key string) (Attributes, error)
}
