package sec

import (
	"github.com/arya-analytics/x/gorp"
	"github.com/google/uuid"
)

type User struct {
	Key      uuid.UUID
	Username string
}

func (u User) Attributes() AccessAttributes { return u.Attrs }

func (u User) GorpKey() uuid.UUID { return u.Key }

func (u User) SetOptions() []interface{} { return nil }

func (u *User) maybeGenerateKey() {
	if u.Key == uuid.Nil {
		u.Key = uuid.New()
	}
}

type UserService struct {
	DB *gorp.DB
}

func (svc *UserService) Retrieve(key uuid.UUID) (u User, err error) {
	return u, gorp.NewRetrieve[uuid.UUID, User]().WhereKeys(key).Entry(&u).Exec(svc.DB)
}

func (svc *UserService) RetrieveByUsername(username string) (u User,
	err error) {
	return u, gorp.NewRetrieve[uuid.UUID, User]().
		Where(func(u User) bool { return u.Username == username }).
		Entry(&u).
		Exec(svc.DB)
}

func (svc *UserService) Register(u *User) (err error) {
	u.Key = uuid.New()
	return gorp.NewCreate[uuid.UUID, User]().Entry(u).Exec(svc.DB)
}

func (svc *UserService) Save(u User) error {
	return gorp.NewCreate[uuid.UUID, User]().Entry(&u).Exec(svc.DB)
}
