package crypto_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writePEM(t *testing.T, name, blockType string, der []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}), 0o600))
	return path
}

func TestLoadPrivateKey_PKCS1AndPKCS8(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pkcs1 := writePEM(t, "pkcs1.pem", "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key))
	loaded, err := crypto.LoadPrivateKey(pkcs1)
	require.NoError(t, err)
	assert.True(t, key.Equal(loaded))

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	pkcs8 := writePEM(t, "pkcs8.pem", "PRIVATE KEY", der)
	loaded, err = crypto.LoadPrivateKey(pkcs8)
	require.NoError(t, err)
	assert.True(t, key.Equal(loaded))
}

func TestLoadPrivateKey_Errors(t *testing.T) {
	t.Run("файла нет", func(t *testing.T) {
		_, err := crypto.LoadPrivateKey(filepath.Join(t.TempDir(), "absent.pem"))
		assert.ErrorContains(t, err, "failed to read private key file")
	})

	t.Run("не PEM", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "garbage.pem")
		require.NoError(t, os.WriteFile(path, []byte("not a pem"), 0o600))

		_, err := crypto.LoadPrivateKey(path)
		assert.ErrorContains(t, err, "failed to decode PEM block")
	})

	t.Run("битый DER", func(t *testing.T) {
		path := writePEM(t, "broken.pem", "PRIVATE KEY", []byte{1, 2, 3})

		_, err := crypto.LoadPrivateKey(path)
		assert.ErrorContains(t, err, "failed to parse private key")
	})

	t.Run("не RSA", func(t *testing.T) {
		ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		der, err := x509.MarshalPKCS8PrivateKey(ecKey)
		require.NoError(t, err)

		_, err = crypto.LoadPrivateKey(writePEM(t, "ec.pem", "PRIVATE KEY", der))
		assert.ErrorContains(t, err, "is not an RSA key")
	})
}

func TestLoadPublicKey_PKIXAndPKCS1(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)

	loaded, err := crypto.LoadPublicKey(writePEM(t, "pkix.pem", "PUBLIC KEY", der))
	require.NoError(t, err)
	assert.True(t, key.PublicKey.Equal(loaded))

	loaded, err = crypto.LoadPublicKey(
		writePEM(t, "pkcs1.pem", "RSA PUBLIC KEY", x509.MarshalPKCS1PublicKey(&key.PublicKey)),
	)
	require.NoError(t, err)
	assert.True(t, key.PublicKey.Equal(loaded))
}

func TestLoadPublicKey_Errors(t *testing.T) {
	t.Run("файла нет", func(t *testing.T) {
		_, err := crypto.LoadPublicKey(filepath.Join(t.TempDir(), "absent.pem"))
		assert.ErrorContains(t, err, "failed to read public key file")
	})

	t.Run("не PEM", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "garbage.pem")
		require.NoError(t, os.WriteFile(path, []byte("nope"), 0o600))

		_, err := crypto.LoadPublicKey(path)
		assert.ErrorContains(t, err, "failed to decode PEM block")
	})

	t.Run("битый DER", func(t *testing.T) {
		_, err := crypto.LoadPublicKey(writePEM(t, "broken.pem", "PUBLIC KEY", []byte{1, 2, 3}))
		assert.ErrorContains(t, err, "failed to parse public key")
	})

	t.Run("не RSA", func(t *testing.T) {
		ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		der, err := x509.MarshalPKIXPublicKey(&ecKey.PublicKey)
		require.NoError(t, err)

		_, err = crypto.LoadPublicKey(writePEM(t, "ec.pem", "PUBLIC KEY", der))
		assert.ErrorContains(t, err, "is not an RSA key")
	})
}
