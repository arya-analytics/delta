package sec

import "github.com/arya-analytics/delta/pkg/sec/password"

type Authenticator interface {
	Authenticate(user string, pass password.Raw) error
	Register(user string, pass password.Raw) error
	UpdateUsername(user, newUser string, pass password.Raw) error
	UpdatePassword(user string, pass, newPass password.Raw) error
}
