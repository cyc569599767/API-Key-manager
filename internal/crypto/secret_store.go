package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"strings"

	"keymanager/internal/storage"
)

type SecretStore struct {
	key []byte
}

func NewSecretStore() (*SecretStore, error) {
	paths, err := storage.AppPaths()
	if err != nil {
		return nil, err
	}
	keyPath := paths.SecretKey
	key, err := os.ReadFile(keyPath)
	if errors.Is(err, os.ErrNotExist) {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(key)), 0o600); err != nil {
			return nil, err
		}
		return &SecretStore{key: key}, nil
	}
	if err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(key)))
	if err != nil {
		return nil, err
	}
	if len(decoded) != 32 {
		return nil, errors.New("invalid secret key length")
	}
	return &SecretStore{key: decoded}, nil
}

func (s *SecretStore) EncryptSecret(plaintext string) ([]byte, []byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return gcm.Seal(nil, nonce, []byte(plaintext), nil), nonce, nil
}

func (s *SecretStore) DecryptSecret(ciphertext []byte, nonce []byte) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (s *SecretStore) FingerprintSecret(plaintext string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(plaintext))
	return hex.EncodeToString(mac.Sum(nil))
}

func MaskSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	prefixLen := min(7, len(secret))
	suffixLen := 4
	if len(secret) <= prefixLen+suffixLen {
		return strings.Repeat("•", len(secret))
	}
	return secret[:prefixLen] + "••••••••••••" + secret[len(secret)-suffixLen:]
}
