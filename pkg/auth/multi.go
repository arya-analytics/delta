package auth

import (
	"github.com/arya-analytics/delta/pkg/password"
	"github.com/arya-analytics/x/gorp"
	"github.com/cockroachdb/errors"
)

// MultiAuthenticator implements the Authenticator interface by wrapping a set of
// exiting Authenticator(s). This is useful for combining multiple Authentication
// sources into a single interface. Authenticator(s) are executed in order,
// and the first Authenticator to succeeds (i.e. non-nil error) is used for the operation.
type MultiAuthenticator []Authenticator

// Authenticate implements the Authenticator interface.
func (a MultiAuthenticator) Authenticate(creds InsecureCredentials) error {
	for _, auth := range a {
		if err := auth.Authenticate(creds); !errors.Is(err, InvalidCredentials) {
			return nil
		}
	}
	return InvalidCredentials
}

// Register implements the Authenticator interface.
func (a MultiAuthenticator) Register(txn gorp.Txn, creds InsecureCredentials) error {
	for _, auth := range a {
		if err := auth.Register(txn, creds); err == nil {
			return nil
		}
	}
	return RegistrationFailed
}

// UpdateUsername implements the Authenticator interface.
func (a MultiAuthenticator) UpdateUsername(txn gorp.Txn, creds InsecureCredentials,
	newUser string) error {
	for _, auth := range a {
		if err := auth.UpdateUsername(txn, creds, newUser); err == nil {
			return nil
		}
	}
	return errors.New("[auth] - failed to update username")
}

// UpdatePassword implements the Authenticator interface.
func (a MultiAuthenticator) UpdatePassword(
	txn gorp.Txn,
	creds InsecureCredentials,
	newPass password.Raw,
) error {
	for _, auth := range a {
		if err := auth.UpdatePassword(txn, creds, newPass); err == nil {
			return nil
		}
	}
	return errors.New("[auth] - failed to update password")
}
