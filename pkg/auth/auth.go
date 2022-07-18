package auth

import (
	"github.com/arya-analytics/delta/pkg/auth/password"
	"github.com/arya-analytics/x/gorp"
)

// Authenticator is an interface for validating the identity of a particular entity (
// i.e. they are who they say they are).
type Authenticator interface {
	// Authenticate validates the identity of the entity with the given credentials.
	// If the credentials are invalid, an InvalidCredentials error is returned.
	Authenticate(creds InsecureCredentials) error
	// Register registers the given credentials in the authenticator.
	// If the Authenticator uses the Node's local storage, they can use the provided
	// txn to perform the registration.
	Register(txn gorp.Txn, creds InsecureCredentials) error
	// UpdateUsername updates the username of the given credentials.
	// If the Authenticator uses the Node's local storage, they can use the provided
	// txn to perform the update.
	UpdateUsername(txn gorp.Txn, creds InsecureCredentials, newUser string) error
	// UpdatePassword updates the password of the given credentials.
	// If the Authenticator uses the Node's local storage, they can use the provided
	// txn to perform the update.
	UpdatePassword(txn gorp.Txn, creds InsecureCredentials, newPass password.Raw) error
}
