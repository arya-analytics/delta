package fiber

import (
	"github.com/arya-analytics/delta/pkg/sec"
	"github.com/arya-analytics/delta/pkg/sec/token"
)

type Config struct {
	Authenticator sec.Authenticator
	Users         *sec.UserService
	Token         *token.Service
}
