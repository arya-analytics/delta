package sec

import "github.com/arya-analytics/delta/pkg/sec/password"

type Credentials struct {
	Username string       `json:"username"`
	Password password.Raw `json:"password"`
}

type Authenticator interface {
	Authenticate(creds Credentials) error
	Register(creds Credentials) error
	UpdateUsername(creds Credentials, newUser string) error
	UpdatePassword(creds Credentials, newPass password.Raw) error
}
