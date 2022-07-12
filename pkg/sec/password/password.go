package password

import (
	"github.com/cockroachdb/errors"
	"golang.org/x/crypto/bcrypt"
)

type Raw string

const (
	hashCost = bcrypt.DefaultCost
)

func (r Raw) Hash() (h Hashed, err error) {
	for _, hasher := range Hashers {
		h, err = hasher.Hash(r)
		if err == nil {
			return h, nil
		}
	}
	return h, errors.Wrap(err, "failed to hash password")
}

type Hashed []byte

func (h Hashed) Validate(r Raw) (err error) {
	for _, hasher := range Hashers {
		err = hasher.Compare(r, h)
		if err == nil {
			return nil
		}
	}
	return errors.Wrap(err, "invalid credentials")
}
