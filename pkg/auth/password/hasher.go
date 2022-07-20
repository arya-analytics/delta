package password

import "golang.org/x/crypto/bcrypt"

// Hashers is a list of hashers that delta uses to encrypt and validate passwords.
// The Hashers are tried in order. The first Hasher that returns a non-nil error
// is used. This value should generally remain unmodified unless the cluster is tailored
// to specific user needs.
var Hashers = DefaultHashers()

// DefaultHashers returns a list of default hashers that delta uses to encrypt and
// validate/passwords.
func DefaultHashers() []Hasher {
	return []Hasher{
		BcryptHasher{},
	}
}

// Hasher hashes and compares passwords against a hash.
type Hasher interface {
	// Hash hashes a Raw password. Returns an error if the password cannot be hashed.
	Hash(pwd Raw) (Hashed, error)
	// Compare compares a Raw password against a Hashed password. Returns an error
	// if pwd does not match the hash.
	Compare(pwd Raw, hash Hashed) error
}

// BcryptHasher is a Hasher that uses the bcrypt library to hash and compare passwords.
type BcryptHasher struct{}

const hashCost = bcrypt.DefaultCost

// Hash implements the Hasher interface.
func (BcryptHasher) Hash(pwd Raw) (Hashed, error) {
	return bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
}

// Compare implements the Hasher interface.
func (BcryptHasher) Compare(pwd Raw, hash Hashed) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(pwd))
}
