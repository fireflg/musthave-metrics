package agent_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/agent"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readGzip(t *testing.T, r io.Reader) []byte {
	t.Helper()

	zr, err := gzip.NewReader(r)
	require.NoError(t, err)
	defer zr.Close()

	data, err := io.ReadAll(zr)
	require.NoError(t, err)
	return data
}

func TestReporter_ReportSendsGzippedBatch(t *testing.T) {
	type payloadItem struct {
		ID    string   `json:"id"`
		MType string   `json:"type"`
		Delta *int64   `json:"delta"`
		Value *float64 `json:"value"`
	}

	var items []payloadItem
	var encoding, contentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoding = r.Header.Get("Content-Encoding")
		contentType = r.Header.Get("Content-Type")
		require.NoError(t, json.Unmarshal(readGzip(t, r.Body), &items))
	}))
	defer srv.Close()

	reporter := agent.NewReporter(srv.URL, "", nil)

	err := reporter.Report(context.Background(), agent.Metrics{"Alloc": 12.5, "PollCount": 3})
	require.NoError(t, err)

	assert.Equal(t, "gzip", encoding)
	assert.Equal(t, "application/json", contentType)
	require.Len(t, items, 2)

	byID := map[string]payloadItem{}
	for _, item := range items {
		byID[item.ID] = item
	}

	require.Contains(t, byID, "Alloc")
	assert.Equal(t, "gauge", byID["Alloc"].MType)
	assert.Equal(t, 12.5, *byID["Alloc"].Value)

	require.Contains(t, byID, "PollCount")
	assert.Equal(t, "counter", byID["PollCount"].MType)
	assert.Equal(t, int64(3), *byID["PollCount"].Delta)
}

func TestReporter_ReportSignsPayload(t *testing.T) {
	const secret = "secret-key"

	var gotHash string
	var gotBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHash = r.Header.Get("HashSHA256")
		gotBody = readGzip(t, r.Body)
	}))
	defer srv.Close()

	reporter := agent.NewReporter(srv.URL, secret, nil)
	require.NoError(t, reporter.Report(context.Background(), agent.Metrics{"Alloc": 1}))

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(gotBody)
	assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), gotHash, "подпись считается от несжатого payload")
}

func TestReporter_ReportEncryptsPayload(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
	}))
	defer srv.Close()

	reporter := agent.NewReporter(srv.URL, "", &key.PublicKey)
	require.NoError(t, reporter.Report(context.Background(), agent.Metrics{"Alloc": 1}))

	decrypted, err := crypto.Decrypt(key, body)
	require.NoError(t, err)

	var items []map[string]any
	require.NoError(t, json.Unmarshal(readGzip(t, bytes.NewReader(decrypted)), &items))
	require.Len(t, items, 1)
	assert.Equal(t, "Alloc", items[0]["id"])
}

func TestReporter_ReportSetsXRealIPHeader(t *testing.T) {
	var gotHeader string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Real-IP")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	reporter := agent.NewReporter(srv.URL, "", nil)

	err := reporter.Report(context.Background(), agent.Metrics{"Alloc": 1.0})
	require.NoError(t, err)
	assert.NotEmpty(t, gotHeader)
}

func TestReporter_NoXRealIPForUnresolvableHost(t *testing.T) {
	reporter := agent.NewReporter("http://", "", nil)
	assert.NotNil(t, reporter)
}

func TestReporter_ReportBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	reporter := agent.NewReporter(srv.URL, "", nil)

	err := reporter.Report(context.Background(), agent.Metrics{"Alloc": 1})
	assert.ErrorContains(t, err, "bad status")
}

func TestReporter_ReportCanceledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := agent.NewReporter(srv.URL, "", nil).Report(ctx, agent.Metrics{"Alloc": 1})
	assert.Error(t, err)
}

func TestReporter_WaitServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	assert.NoError(t, agent.NewReporter(srv.URL, "", nil).WaitServer(context.Background()))
}

func TestReporter_WaitServerBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer srv.Close()

	err := agent.NewReporter(srv.URL, "", nil).WaitServer(context.Background())
	assert.ErrorContains(t, err, "bad status")
}

func TestReporter_WaitServerCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := agent.NewReporter("http://127.0.0.1:1", "", nil).WaitServer(ctx)
	assert.Error(t, err)
}
