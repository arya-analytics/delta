package rbac

import (
	"github.com/arya-analytics/x/gorp"
	kvx "github.com/arya-analytics/x/kv"
	"strings"
)

type Storage interface {
	get(p policy) (bool, error)
	set(p policy) error
}

func NewKVStorage(kve kvx.KV) Storage {
	return &kvStorage{db: gorp.Wrap(kve)}
}

type policy struct {
	subject, object, action string
}

func (p policy) GorpKey() string {
	return strings.Join([]string{p.subject, p.object, p.action}, "/")
}

func (p policy) SetOptions() []interface{} { return nil }

type kvStorage struct {
	db *gorp.DB
}

func (kv *kvStorage) get(p policy) (bool, error) {
	return gorp.NewRetrieve[string, policy]().WhereKeys(p.GorpKey()).Exists(kv.db)
}

func (kv *kvStorage) set(p policy) error {
	return gorp.NewCreate[string, policy]().Entry(&p).Exec(kv.db)
}
