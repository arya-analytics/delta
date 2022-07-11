package sec

import (
	"github.com/arya-analytics/x/gorp"
	"github.com/google/uuid"
)

type User struct {
	Key      uuid.UUID
	Username string
	Attrs    AccessAttributes
}

func (u User) Attributes() AccessAttributes { return u.Attrs }

func (u User) GorpKey() uuid.UUID { return u.Key }

func (u User) SetOptions() []interface{} { return nil }

func (u *User) maybeGenerateKey() {
	if u.Key == uuid.Nil {
		u.Key = uuid.New()
	}
}

func (u *User) Save(db *gorp.DB) error {
	u.maybeGenerateKey()
	return gorp.NewCreate[uuid.UUID, User]().Entry(u).Exec(db)
}

func RetrieveUser(db *gorp.DB, key uuid.UUID) (u User, err error) {
	return u, gorp.NewRetrieve[uuid.UUID, User]().WhereKeys(key).Entry(&u).Exec(db)
}

func RetrieveUserByUsername(db *gorp.DB, username string) (u User, err error) {
	return u, gorp.NewRetrieve[uuid.UUID, User]().
		Where(func(u User) bool { return u.Username == username }).
		Entry(&u).
		Exec(db)
}

func RetrieveUsers(db *gorp.DB, keys ...uuid.UUID) (users []User, err error) {
	return users, gorp.NewRetrieve[uuid.UUID, User]().WhereKeys(keys...).Entries(&users).Exec(db)
}

func SaveUsers(db *gorp.DB, users []User) error {
	return gorp.NewCreate[uuid.UUID, User]().Entries(&users).Exec(db)
}
