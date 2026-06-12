package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_WithValidSecret(t *testing.T) {
	c, err := New("a-strong-secret-for-testing")
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestNew_WithEmptySecret(t *testing.T) {
	_, err := New("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret must not be empty")
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	c, err := New("test-secret-key-for-encryption")
	require.NoError(t, err)

	plaintext := "hello, world!"
	encrypted, err := c.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := c.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_ProducesDifferentCiphertext(t *testing.T) {
	c, err := New("test-secret-key")
	require.NoError(t, err)

	plaintext := "same input"
	enc1, err := c.Encrypt(plaintext)
	require.NoError(t, err)

	enc2, err := c.Encrypt(plaintext)
	require.NoError(t, err)

	// Same plaintext should produce different ciphertext due to random nonce.
	assert.NotEqual(t, enc1, enc2)
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	c, err := New("test-secret")
	require.NoError(t, err)

	encrypted, err := c.Encrypt("secret data")
	require.NoError(t, err)

	// Tamper with the ciphertext.
	tampered := encrypted[:len(encrypted)-2] + "XX"

	_, err = c.Decrypt(tampered)
	assert.Error(t, err)
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	c, err := New("test-secret")
	require.NoError(t, err)

	_, err = c.Decrypt("not-valid-base64!!!")
	assert.Error(t, err)
}

func TestDecrypt_WrongKey(t *testing.T) {
	c1, err := New("secret-one")
	require.NoError(t, err)

	c2, err := New("secret-two")
	require.NoError(t, err)

	encrypted, err := c1.Encrypt("sensitive")
	require.NoError(t, err)

	_, err = c2.Decrypt(encrypted)
	assert.Error(t, err)
}

func TestEncryptDecrypt_EmptyString(t *testing.T) {
	c, err := New("test-secret")
	require.NoError(t, err)

	encrypted, err := c.Encrypt("")
	require.NoError(t, err)

	decrypted, err := c.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "", decrypted)
}

func TestEncryptDecrypt_LongString(t *testing.T) {
	c, err := New("test-secret")
	require.NoError(t, err)

	long := strings.Repeat("abcdefghij", 1024) // 10 KB

	encrypted, err := c.Encrypt(long)
	require.NoError(t, err)

	decrypted, err := c.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, long, decrypted)
}
