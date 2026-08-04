// Package crypto предоставляет гибридное RSA+AES шифрование сообщений
// от агента к серверу и загрузку ключей из PEM-файлов.
package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadPublicKey читает PEM-файл и возвращает RSA публичный ключ.
// Поддерживаются форматы PKIX ("PUBLIC KEY") и PKCS1 ("RSA PUBLIC KEY").
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", path)
	}

	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("public key in %s is not an RSA key", path)
		}
		return rsaPub, nil
	}

	rsaPub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key from %s: %w", path, err)
	}
	return rsaPub, nil
}

// LoadPrivateKey читает PEM-файл и возвращает RSA приватный ключ.
// Поддерживаются форматы PKCS1 ("RSA PRIVATE KEY") и PKCS8 ("PRIVATE KEY").
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", path)
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key from %s: %w", path, err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key in %s is not an RSA key", path)
	}
	return rsaKey, nil
}
