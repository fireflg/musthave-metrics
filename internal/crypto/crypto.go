package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

const aesKeySize = 32

// Encrypt шифрует plaintext
func Encrypt(pub *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	aesKey := make([]byte, aesKeySize)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	encKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt AES key: %w", err)
	}

	result := make([]byte, 0, 2+len(encKey)+len(nonce)+len(ciphertext))
	keyLen := make([]byte, 2)
	binary.BigEndian.PutUint16(keyLen, uint16(len(encKey)))

	result = append(result, keyLen...)
	result = append(result, encKey...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// Decrypt расшифровывает данные, зашифрованные функцией Encrypt.
func Decrypt(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("encrypted data too short")
	}

	keyLen := int(binary.BigEndian.Uint16(data[:2]))
	data = data[2:]

	if len(data) < keyLen {
		return nil, fmt.Errorf("encrypted data too short for encrypted key")
	}

	encKey := data[:keyLen]
	rest := data[keyLen:]

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt AES key: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(rest) < nonceSize {
		return nil, fmt.Errorf("encrypted data too short for nonce")
	}

	nonce, ciphertext := rest[:nonceSize], rest[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt payload: %w", err)
	}

	return plaintext, nil
}
