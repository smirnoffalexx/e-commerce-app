package security

import (
	"fmt"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

type BcryptHasher struct {
	cost      int
	dummyOnce sync.Once
	dummyHash []byte
}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{cost: bcryptCost}
}

func (h *BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(b), nil
}

func (h *BcryptHasher) Compare(hash, password string) bool {
	if hash == "" {
		_ = bcrypt.CompareHashAndPassword(h.dummy(), []byte(password))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (h *BcryptHasher) dummy() []byte {
	h.dummyOnce.Do(func() {
		h.dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer"), h.cost)
	})
	return h.dummyHash
}
