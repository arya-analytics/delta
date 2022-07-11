package auth

import "github.com/arya-analytics/delta/pkg/sec/password"

type usernamePasswordPair struct {
	Username string
	Password password.Hashed
}
