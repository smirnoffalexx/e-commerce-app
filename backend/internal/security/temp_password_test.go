package security

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/domain"
)

func TestTempPasswordGenerator(t *testing.T) {
	gen := NewTempPasswordGenerator()
	seen := make(map[string]bool, 1000)

	for range 1000 {
		p, err := gen.Generate()
		require.NoError(t, err)

		assert.Len(t, p, tempPasswordLen)
		for _, r := range p {
			assert.True(t, strings.ContainsRune(tempPasswordAlphabet, r), "unexpected rune %q", r)
		}
		assert.NoError(t, domain.ValidatePassword("password", p, ""), "temporary password must satisfy the password policy")
		assert.False(t, seen[p], "duplicate temporary password")
		seen[p] = true
	}

	assert.NotContains(t, tempPasswordAlphabet, "0")
	assert.NotContains(t, tempPasswordAlphabet, "O")
	assert.NotContains(t, tempPasswordAlphabet, "l")
	assert.NotContains(t, tempPasswordAlphabet, "1")
	assert.NotContains(t, tempPasswordAlphabet, "I")
}
