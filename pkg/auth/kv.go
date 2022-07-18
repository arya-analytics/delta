package auth

import (
	"github.com/arya-analytics/delta/pkg/password"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/query"
	"github.com/cockroachdb/errors"
)

// KV is a simple key-value backed Authenticator. It saves data to the provided
// gorp DB. It's important to note that all gorp.txn(s) provided to the Authenticator
// interface must be spawned from the same gorp DB.
type KV struct{ DB *gorp.DB }

// Authenticate implements the sec.Authenticator interface.
func (db *KV) Authenticate(user string, pass password.Raw) error {
	up, err := db.retrieve(db.DB, user)
	if err != nil {
		return err
	}
	return up.Password.Validate(pass)
}

// Register implements the sec.Authenticator interface.
func (db *KV) Register(
	txn gorp.Txn,
	user string,
	pass password.Raw,
) error {
	exists, err := db.exists(txn, user)
	if exists {
		return errors.Wrap(query.UniqueViolation, "[auth] - username already exists")
	}
	if err != nil {
		return err
	}
	hash, err := pass.Hash()
	if err != nil {
		return err
	}
	return db.set(txn, user, hash)
}

// UpdateUsername implements the sec.Authenticator interface.
func (db *KV) UpdateUsername(
	txn gorp.Txn,
	user, newUser string,
	pass password.Raw,
) error {
	up, err := db.retrieve(txn, user)
	if err != nil {
		return err
	}
	if err := up.Password.Validate(pass); err != nil {
		return err
	}
	up.Username = newUser
	return db.set(txn, newUser, up.Password)
}

// UpdatePassword implements the sec.Authenticator interface.
func (db *KV) UpdatePassword(
	txn gorp.Txn,
	user string,
	pass, newPass password.Raw,
) error {
	up, err := db.retrieve(txn, user)
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
	return db.set(txn, user, hash)
}

func (db *KV) exists(txn gorp.Txn, user string) (bool, error) {
	return gorp.NewRetrieve[string, SecureCredentials]().
		WhereKeys(user).
		Exists(txn)
}

func (db *KV) retrieve(txn gorp.Txn, user string) (SecureCredentials, error) {
	var creds SecureCredentials
	return creds, gorp.NewRetrieve[string, SecureCredentials]().
		WhereKeys(user).
		Entry(&creds).Exec(txn)
}

func (db *KV) set(txn gorp.Txn, user string, password password.Hashed) error {
	return gorp.NewCreate[string, SecureCredentials]().
		Entry(&SecureCredentials{Username: user, Password: password}).
		Exec(txn)
}
