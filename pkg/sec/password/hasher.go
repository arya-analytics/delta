package password

import "golang.org/x/crypto/bcrypt"

var Hashers = DefaultHashers()

func DefaultHashers() []Hasher {
	return []Hasher{
		BcryptHasher{},
	}
}

type Hasher interface {
	Hash(pwd Raw) (Hashed, error)
	Compare(pwd Raw, hash Hashed) error
}

type BcryptHasher struct{}

func (BcryptHasher) Hash(pwd Raw) (Hashed, error) {
	return bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
}

func (BcryptHasher) Compare(pwd Raw, hash Hashed) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(pwd))
}
