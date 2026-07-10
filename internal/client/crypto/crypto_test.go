package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timofeevav/gophkeeper/internal/client/crypto"
)

func TestDeriveKey(t *testing.T) {
	key := crypto.DeriveKey("masterpassword", "user-id-123")
	assert.Len(t, key, 32)

	key2 := crypto.DeriveKey("masterpassword", "user-id-123")
	assert.Equal(t, key, key2, "key derivation must be deterministic")

	key3 := crypto.DeriveKey("otherpassword", "user-id-123")
	assert.NotEqual(t, key, key3, "different passwords must produce different keys")

	key4 := crypto.DeriveKey("masterpassword", "other-user-id")
	assert.NotEqual(t, key, key4, "different user IDs must produce different keys")
}

func TestEncryptDecrypt(t *testing.T) {
	key := crypto.DeriveKey("masterpassword", "user-id-123")
	plaintext := []byte(`{"login":"user","password":"secret"}`)

	ciphertext, err := crypto.Encrypt(key, plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := crypto.Decrypt(key, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_UniqueNonce(t *testing.T) {
	key := crypto.DeriveKey("masterpassword", "user-id-123")
	plaintext := []byte("same plaintext")

	ct1, err := crypto.Encrypt(key, plaintext)
	require.NoError(t, err)
	ct2, err := crypto.Encrypt(key, plaintext)
	require.NoError(t, err)

	assert.NotEqual(t, ct1, ct2, "each encryption must use a unique nonce")
}

func TestDecrypt_WrongKey(t *testing.T) {
	key := crypto.DeriveKey("correctpassword", "user-id-123")
	wrongKey := crypto.DeriveKey("wrongpassword", "user-id-123")

	ciphertext, err := crypto.Encrypt(key, []byte("secret data"))
	require.NoError(t, err)

	_, err = crypto.Decrypt(wrongKey, ciphertext)
	assert.Error(t, err)
}

func TestDecrypt_TamperedData(t *testing.T) {
	key := crypto.DeriveKey("masterpassword", "user-id-123")
	ciphertext, err := crypto.Encrypt(key, []byte("secret data"))
	require.NoError(t, err)

	ciphertext[len(ciphertext)-1] ^= 0xff

	_, err = crypto.Decrypt(key, ciphertext)
	assert.Error(t, err)
}

func TestDecrypt_TooShort(t *testing.T) {
	key := crypto.DeriveKey("masterpassword", "user-id-123")
	_, err := crypto.Decrypt(key, []byte("short"))
	assert.Error(t, err)
}
