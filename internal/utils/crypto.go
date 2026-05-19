package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// EncryptField encrypts a string field using AES-256-GCM.
// Returns base64url-encoded string safe to store in DB varchar/text columns.
func EncryptField(plaintext string, keyHex string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key, err := hexToBytes32(keyHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// DecryptField decrypts a base64url-encoded AES-256-GCM ciphertext.
// Returns original plaintext. If input is not encrypted (legacy plain text),
// returns the original value so existing data remains accessible.

func DecryptField(encoded string, keyHex string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	key, err := hexToBytes32(keyHex)
	if err != nil {
		return encoded, nil // graceful fallback for plain text
	}

	current := encoded
	// Lakukan loop dekripsi maksimal 3 kali untuk memulihkan double-encryption otomatis
	for i := 0; i < 3; i++ {
		data, err := base64.URLEncoding.DecodeString(current)
		if err != nil {
			break // Bukan base64 URL safe, kembalikan current
		}

		block, err := aes.NewCipher(key)
		if err != nil {
			break
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			break
		}

		if len(data) < gcm.NonceSize() {
			break
		}

		nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			break // Gagal dekripsi (bukan ciphertext valid untuk key ini), selesai
		}

		current = string(plaintext)
	}

	return current, nil
}

// hexToBytes32 decodes a 64-char hex string to 32 bytes for AES-256.
func hexToBytes32(keyHex string) ([]byte, error) {
	if len(keyHex) != 64 {
		return nil, errors.New("encryption key must be 64 hex chars (32 bytes)")
	}
	key := make([]byte, 32)
	for i := 0; i < 32; i++ {
		b, err := hexByte(keyHex[i*2], keyHex[i*2+1])
		if err != nil {
			return nil, errors.New("invalid hex in encryption key")
		}
		key[i] = b
	}
	return key, nil
}

func hexByte(hi, lo byte) (byte, error) {
	h, err := hexNibble(hi)
	if err != nil {
		return 0, err
	}
	l, err := hexNibble(lo)
	if err != nil {
		return 0, err
	}
	return (h << 4) | l, nil
}

func hexNibble(c byte) (byte, error) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', nil
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, nil
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, nil
	default:
		return 0, errors.New("invalid hex char")
	}
}
