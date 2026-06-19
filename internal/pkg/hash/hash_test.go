package hash_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/go-api-template/internal/pkg/hash"
)

func TestHashPassword(t *testing.T) {
	plain := "MySecretPassword123!"
	hashed, err := hash.Password(plain)
	require.NoError(t, err)
	assert.NotEmpty(t, hashed)
	assert.NotEqual(t, plain, hashed)
}

func TestComparePassword_Success(t *testing.T) {
	plain := "MySecretPassword123!"
	hashed, _ := hash.Password(plain)
	assert.True(t, hash.ComparePassword(hashed, plain))
}

func TestComparePassword_WrongPassword(t *testing.T) {
	hashed, _ := hash.Password("correct")
	assert.False(t, hash.ComparePassword(hashed, "wrong"))
}

func TestHashIsUnique(t *testing.T) {
	plain := "same-password"
	h1, _ := hash.Password(plain)
	h2, _ := hash.Password(plain)
	assert.NotEqual(t, h1, h2)
}
