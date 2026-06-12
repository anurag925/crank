package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Crypto encrypts and decrypts byte strings using AES-256-GCM. The 32-byte
// key is derived from a human-readable secret via SHA-256 so callers only need
// to supply a sufficiently long passphrase (stored in .env as CRYPTO_SECRET).
type Crypto struct {
	aead cipher.AEAD
}

// New creates a Crypto instance from the supplied secret. The secret is hashed
// with SHA-256 to produce a 32-byte AES key, so any non-empty string works.
//
// For production use, supply at least 32 bytes of entropy via the
// CRYPTO_SECRET environment variable.
func New(secret string) (*Crypto, error) {
	if secret == "" {
		return nil, errors.New("crypto: secret must not be empty")
	}

	key := sha256.Sum256([]byte(secret))

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}

	return &Crypto{aead: aead}, nil
}

// Encrypt serialises plaintext into a base64-url string containing the random
// nonce prepended to the AES-256-GCM ciphertext. The nonce is unique per call
// so the same plaintext produces different ciphertext each time.
func (c *Crypto) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: generate nonce: %w", err)
	}

	// Seal appends the encrypted payload to the nonce so we can extract both
	// from a single byte slice during decryption.
	payload := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

// Decrypt decodes a base64-url string produced by Encrypt and returns the
// original plaintext. Returns an error if the ciphertext is tampered with or
// the wrong secret was used.
func (c *Crypto) Decrypt(encoded string) (string, error) {
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: invalid encoding: %w", err)
	}

	nonceSize := c.aead.NonceSize()
	if len(payload) < nonceSize {
		return "", errors.New("crypto: ciphertext too short")
	}

	nonce, ciphertext := payload[:nonceSize], payload[nonceSize:]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: decrypt: %w", err)
	}

	return string(plaintext), nil
}
