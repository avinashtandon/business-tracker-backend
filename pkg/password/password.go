// Package password provides bcrypt-based password hashing and verification.
//
// Choice: bcrypt over argon2.
// Bcrypt is chosen for its wide ecosystem support, built-in salt, and
// adaptive cost factor. Cost 12 is a good balance between security and
// performance (~250ms on modern hardware), making brute-force attacks
// computationally expensive. Argon2id would be preferred for new systems
// but bcrypt is the safer default given broader library support.
package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// Hash returns a bcrypt hash of the plaintext password.
func Hash(plaintext string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// Verify checks whether plaintext matches the stored bcrypt hash.
// Returns nil on match, ErrMismatch on mismatch, or another error on failure.
func Verify(hash, plaintext string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrMismatch
	}
	return err
}

// ErrMismatch is returned when the password does not match the hash.
var ErrMismatch = errors.New("password does not match")
