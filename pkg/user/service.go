package user

import (
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/x/gorp"
	"github.com/google/uuid"
)

type Service struct {
	db        *gorp.DB
	resources *ontology.Service
}

func (s *Service) Retrieve(key uuid.UUID) (User, error) {
	var u User
	return u, gorp.NewRetrieve[uuid.UUID, User]().WhereKeys(key).Entry(&u).Exec(s.db)
}

func (s *Service) RetrieveByUsername(username string) (User, error) {
	var u User
	return u, gorp.NewRetrieve[uuid.UUID, User]().
		Where(func(u User) bool { return u.Username == username }).
		Entry(&u).
		Exec(s.db)
}

func (s *Service) Create(txn gorp.Txn, u *User) error {
	if u.Key == uuid.Nil {
		u.Key = uuid.New()
	}
	if err := s.resources.NewWriter(txn).DefineResource(ResourceKey(u.
		Key)); err != nil {
		return err
	}
	return gorp.NewCreate[uuid.UUID, User]().Entry(u).Exec(txn)
}
