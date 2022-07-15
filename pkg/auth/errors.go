package auth

import (
	"github.com/arya-analytics/delta/pkg/password"
	"github.com/cockroachdb/errors"
)

var (
	// InvalidCredentials is returned when the credentials for a particular entity
	// are invalid.
	InvalidCredentials = password.Invalid
	// RegistrationFailed is returned when the registration process fails.
	RegistrationFailed = errors.New("[auth] - registration failed")
)
