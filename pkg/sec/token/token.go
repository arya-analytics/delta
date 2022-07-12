package token

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"time"
)

type Service struct {
	Secret     []byte
	Expiration time.Duration
}

func (s *Service) New(issuer uuid.UUID) (string, error) {
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Issuer:    issuer.String(),
		ExpiresAt: time.Now().Add(s.Expiration).Unix(),
	})
	return claims.SignedString(s.Secret)
}

func (s *Service) Validate(token string) (uuid.UUID, error) {
	claims := &jwt.StandardClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return s.Secret, nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(claims.Issuer)
}
