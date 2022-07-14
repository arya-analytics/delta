package auth

import (
	"github.com/arya-analytics/delta/pkg/sec"
	"github.com/arya-analytics/delta/pkg/sec/password"
	"github.com/arya-analytics/x/gorp"
	kvx "github.com/arya-analytics/x/kv"
	"github.com/cockroachdb/errors"
)

type kv struct {
	gorpDB *gorp.DB
}

func NewKV(kve kvx.DB) sec.Authenticator { return &kv{gorpDB: gorp.Wrap(kve)} }

// GorpKey implements the gorp.Entry interface.
func (u usernamePasswordPair) GorpKey() string { return u.Username }

// SetOptions implements the gorp.Entry interface.
func (u usernamePasswordPair) SetOptions() []interface{} { return nil }

// Authenticate implements the sec.Authenticator interface.
func (db *kv) Authenticate(user string, pass password.Raw) error {
	up, err := db.retrieve(user)
	if err != nil {
		return err
	}
	return up.Password.Validate(pass)
}

// Register implements the sec.Authenticator interface.
func (db *kv) Register(user string, pass password.Raw) error {
	exists, err := db.exists(user)
	if exists {
		return errors.New("[kvauth] - user already exists")
	}
	if err != nil {
		return err
	}

	hash, err := pass.Hash()
	if err != nil {
		return err
	}
	return db.set(user, hash)
}

func (db *kv) UpdateUsername(user, newUser string, pass password.Raw) error {
	up, err := db.retrieve(user)
	if err != nil {
		return err
	}
	if err := up.Password.Validate(pass); err != nil {
		return err
	}
	up.Username = newUser
	return db.set(newUser, up.Password)
}

func (db *kv) UpdatePassword(user string, pass, newPass password.Raw) error {
	up, err := db.retrieve(user)
	if err != nil {
		return err
	}
	if err := up.Password.Validate(pass); err != nil {
		return err
	}
	hash, err := newPass.Hash()
	if err != nil {
		return err
	}
	return db.set(user, hash)
}

func (db *kv) exists(user string) (bool, error) {
	return gorp.NewRetrieve[string, usernamePasswordPair]().
		WhereKeys(user).Exists(db.gorpDB)
}

func (db *kv) retrieve(user string) (usernamePasswordPair, error) {
	up := usernamePasswordPair{}
	if err := gorp.NewRetrieve[string, usernamePasswordPair]().
		WhereKeys(user).
		Entry(&up).Exec(db.gorpDB); err != nil {
		return up, err
	}
	return up, nil
}

func (db *kv) set(user string, password password.Hashed) error {
	up := usernamePasswordPair{Username: user, Password: password}
	return gorp.NewCreate[string, usernamePasswordPair]().Entry(&up).Exec(db.gorpDB)
}
