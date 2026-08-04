package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestKeyPair(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func writePEM(t *testing.T, dir, name string, block *pem.Block) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	require.NoError(t, pem.Encode(f, block))
	return path
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := generateTestKeyPair(t)
	plaintext := []byte(`[{"id":"Alloc","type":"gauge","value":123.45}]`)

	ciphertext, err := Encrypt(&key.PublicKey, plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := Decrypt(key, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestDecrypt_CorruptedData(t *testing.T) {
	key := generateTestKeyPair(t)
	plaintext := []byte("some payload")

	ciphertext, err := Encrypt(&key.PublicKey, plaintext)
	require.NoError(t, err)

	corrupted := append([]byte{}, ciphertext...)
	corrupted[len(corrupted)-1] ^= 0xFF

	_, err = Decrypt(key, corrupted)
	assert.Error(t, err)
}

func TestDecrypt_TooShort(t *testing.T) {
	key := generateTestKeyPair(t)

	_, err := Decrypt(key, []byte{0x00})
	assert.Error(t, err)
}

func TestDecrypt_WrongKey(t *testing.T) {
	key := generateTestKeyPair(t)
	otherKey := generateTestKeyPair(t)
	plaintext := []byte("some payload")

	ciphertext, err := Encrypt(&key.PublicKey, plaintext)
	require.NoError(t, err)

	_, err = Decrypt(otherKey, ciphertext)
	assert.Error(t, err)
}

func TestLoadPublicKey_PKIX(t *testing.T) {
	key := generateTestKeyPair(t)
	dir := t.TempDir()

	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)

	path := writePEM(t, dir, "public.pem", &pem.Block{Type: "PUBLIC KEY", Bytes: der})

	pub, err := LoadPublicKey(path)
	require.NoError(t, err)
	assert.Equal(t, key.PublicKey.N, pub.N)
	assert.Equal(t, key.PublicKey.E, pub.E)
}

func TestLoadPrivateKey_PKCS1(t *testing.T) {
	key := generateTestKeyPair(t)
	dir := t.TempDir()

	der := x509.MarshalPKCS1PrivateKey(key)
	path := writePEM(t, dir, "private.pem", &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})

	priv, err := LoadPrivateKey(path)
	require.NoError(t, err)
	assert.Equal(t, key.D, priv.D)
}

func TestLoadPrivateKey_PKCS8(t *testing.T) {
	key := generateTestKeyPair(t)
	dir := t.TempDir()

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	path := writePEM(t, dir, "private.pem", &pem.Block{Type: "PRIVATE KEY", Bytes: der})

	priv, err := LoadPrivateKey(path)
	require.NoError(t, err)
	assert.Equal(t, key.D, priv.D)
}

func TestLoadPublicKey_FileNotFound(t *testing.T) {
	_, err := LoadPublicKey("/nonexistent/path.pem")
	assert.Error(t, err)
}

func TestLoadPrivateKey_InvalidPEM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.pem")
	require.NoError(t, os.WriteFile(path, []byte("not a pem file"), 0o600))

	_, err := LoadPrivateKey(path)
	assert.Error(t, err)
}

func TestEndToEndWithLoadedKeys(t *testing.T) {
	key := generateTestKeyPair(t)
	dir := t.TempDir()

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pubPath := writePEM(t, dir, "public.pem", &pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	privDER := x509.MarshalPKCS1PrivateKey(key)
	privPath := writePEM(t, dir, "private.pem", &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER})

	pub, err := LoadPublicKey(pubPath)
	require.NoError(t, err)
	priv, err := LoadPrivateKey(privPath)
	require.NoError(t, err)

	plaintext := []byte("end-to-end payload")
	ciphertext, err := Encrypt(pub, plaintext)
	require.NoError(t, err)

	decrypted, err := Decrypt(priv, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}
