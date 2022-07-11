package password

import (
	"github.com/cockroachdb/errors"
	"golang.org/x/crypto/bcrypt"
)

type Raw string

const (
	hashCost = bcrypt.DefaultCost
)

func (r Raw) Hash() (Hashed, error) {
	return bcrypt.GenerateFromPassword([]byte(r), bcrypt.DefaultCost)
}

type Hashed []byte

func (h Hashed) Validate(r Raw) error {
	err := bcrypt.CompareHashAndPassword(h, []byte(r))
	if err != nil {
		return errors.Wrap(err, "invalid credentials")
	}
	return nil
}
