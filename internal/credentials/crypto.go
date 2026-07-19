package credentials

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// CurrentKeyVersion is written into new encrypted payloads.
const CurrentKeyVersion = 1

// Box seals and opens credential secrets with a process master key.
type Box struct {
	key []byte
}

// ParseMasterKey derives a 32-byte AES key.
// Accepts raw UTF-8 (SHA-256) or base64-encoded 32-byte material (with optional "base64:" prefix).
func ParseMasterKey(raw string) (*Box, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty credentials master key")
	}
	var key []byte
	if strings.HasPrefix(raw, "base64:") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(raw, "base64:"))
		if err != nil {
			return nil, fmt.Errorf("decode master key: %w", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("base64 master key must decode to 32 bytes, got %d", len(decoded))
		}
		key = decoded
	} else if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) == 32 {
		key = decoded
	} else {
		sum := sha256.Sum256([]byte(raw))
		key = sum[:]
	}
	return &Box{key: key}, nil
}

// Seal encrypts plaintext with AES-256-GCM. Stored as nonce||ciphertext.
func (b *Box) Seal(plaintext string) ([]byte, error) {
	if b == nil || len(b.key) != 32 {
		return nil, fmt.Errorf("credentials master key not configured")
	}
	block, err := aes.NewCipher(b.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// Open decrypts a Seal payload.
func (b *Box) Open(payload []byte) (string, error) {
	if b == nil || len(b.key) != 32 {
		return "", fmt.Errorf("credentials master key not configured")
	}
	if len(payload) == 0 {
		return "", fmt.Errorf("empty encrypted payload")
	}
	block, err := aes.NewCipher(b.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(payload) < ns {
		return "", fmt.Errorf("ciphertext too short")
	}
	plain, err := gcm.Open(nil, payload[:ns], payload[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt credential: %w", err)
	}
	return string(plain), nil
}
