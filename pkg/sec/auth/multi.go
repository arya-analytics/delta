package auth

import "github.com/arya-analytics/delta/pkg/sec"

type MultiAuthenticator []sec.Authenticator

func (m MultiAuthenticator) 
