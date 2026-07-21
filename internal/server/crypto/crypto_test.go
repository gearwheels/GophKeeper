package crypto_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timofeevav/gophkeeper/internal/server/crypto"
)

func TestHashPassword(t *testing.T) {
	hash, err := crypto.HashPassword("strongpassword")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "strongpassword", hash)
}

func TestCheckPassword(t *testing.T) {
	hash, err := crypto.HashPassword("mypassword")
	require.NoError(t, err)

	assert.True(t, crypto.CheckPassword("mypassword", hash))
	assert.False(t, crypto.CheckPassword("wrongpassword", hash))
	assert.False(t, crypto.CheckPassword("", hash))
}

func TestGenerateAndParseToken(t *testing.T) {
	userID := uuid.New()
	secret := "test-jwt-secret-32-bytes-long!!"

	token, err := crypto.GenerateToken(userID, secret, time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	parsedID, err := crypto.ParseToken(token, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, parsedID)
}

func TestParseToken_InvalidSecret(t *testing.T) {
	userID := uuid.New()
	token, err := crypto.GenerateToken(userID, "correct-secret-32-bytes-long!!", time.Hour)
	require.NoError(t, err)

	_, err = crypto.ParseToken(token, "wrong-secret-32-bytes-long!!!")
	assert.Error(t, err)
}

func TestParseToken_Expired(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret-32-bytes-long-here!!"

	token, err := crypto.GenerateToken(userID, secret, -time.Minute)
	require.NoError(t, err)

	_, err = crypto.ParseToken(token, secret)
	assert.Error(t, err)
}

func TestParseToken_Malformed(t *testing.T) {
	_, err := crypto.ParseToken("not.a.valid.token", "secret")
	assert.Error(t, err)
}
