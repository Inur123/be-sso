package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
)

// EncryptAvatar encrypts image bytes using AES-256-GCM.
// Format stored: [4-byte ext len][ext bytes][nonce][ciphertext]
func EncryptAvatar(data []byte, ext string, keyHex string) ([]byte, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil, errors.New("invalid avatar key: must be 32-byte hex")
	}

	block, err := aes.NewCipher(key)
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

	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	// Prepend extension header: [1 byte ext len][ext bytes]
	extBytes := []byte(ext)
	header := append([]byte{byte(len(extBytes))}, extBytes...)

	return append(header, ciphertext...), nil
}

// DecryptAvatar decrypts AES-256-GCM encrypted avatar bytes.
// Returns: decrypted image bytes, original extension (e.g. ".jpg")
func DecryptAvatar(payload []byte, keyHex string) ([]byte, string, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil, "", errors.New("invalid avatar key")
	}

	if len(payload) < 1 {
		return nil, "", errors.New("payload too short")
	}

	// Read extension header
	extLen := int(payload[0])
	if len(payload) < 1+extLen {
		return nil, "", errors.New("payload too short for ext header")
	}
	ext := string(payload[1 : 1+extLen])
	ciphertext := payload[1+extLen:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, "", errors.New("decryption failed: invalid key or corrupted data")
	}

	return plaintext, ext, nil
}
