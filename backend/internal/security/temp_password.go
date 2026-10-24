package security

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	tempPasswordLen      = 16
	tempPasswordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"
)

type TempPasswordGenerator struct{}

func NewTempPasswordGenerator() TempPasswordGenerator {
	return TempPasswordGenerator{}
}

func (TempPasswordGenerator) Generate() (string, error) {
	limit := big.NewInt(int64(len(tempPasswordAlphabet)))
	b := make([]byte, tempPasswordLen)
	for i := range b {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", fmt.Errorf("generate temporary password: %w", err)
		}
		b[i] = tempPasswordAlphabet[n.Int64()]
	}
	return string(b), nil
}
