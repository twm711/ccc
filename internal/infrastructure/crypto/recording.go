package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// RecordingEncryptor encrypts and decrypts recording file data using
// AES-256-GCM, storing the nonce prepended to the ciphertext.
type RecordingEncryptor struct {
	keys map[string][]byte // keyID → 32-byte AES key
}

// NewRecordingEncryptor creates an encryptor with a map of named keys.
func NewRecordingEncryptor(keys map[string][]byte) (*RecordingEncryptor, error) {
	for id, k := range keys {
		if len(k) != 32 {
			return nil, fmt.Errorf("crypto: key %q must be 32 bytes, got %d", id, len(k))
		}
	}
	return &RecordingEncryptor{keys: keys}, nil
}

// Encrypt encrypts plaintext using the given keyID. Returns ciphertext with
// prepended nonce.
func (e *RecordingEncryptor) Encrypt(keyID string, plaintext []byte) ([]byte, error) {
	key, ok := e.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("crypto: unknown key %q", keyID)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: gen nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext (nonce-prepended) using the given keyID.
func (e *RecordingEncryptor) Decrypt(keyID string, ciphertext []byte) ([]byte, error) {
	key, ok := e.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("crypto: unknown key %q", keyID)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("crypto: ciphertext too short")
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}
