package observer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/observer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPObserver_NotifyBatch(t *testing.T) {
	var got observer.AuditEntry
	var contentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	obs := observer.NewHTTPObserver(srv.URL)
	ctx := observer.WithClientIP(context.Background(), "10.0.0.7")
	obs.NotifyBatch(ctx, []string{"Alloc", "PollCount"})

	assert.Equal(t, "application/json", contentType)
	assert.Equal(t, []string{"Alloc", "PollCount"}, got.Metrics)
	assert.Equal(t, "10.0.0.7", got.IPAddress)
	assert.NotZero(t, got.TS)
}

func TestHTTPObserver_Notify(t *testing.T) {
	var got observer.AuditEntry

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
	}))
	defer srv.Close()

	observer.NewHTTPObserver(srv.URL).Notify(context.Background(), "Alloc")

	assert.Equal(t, []string{"Alloc"}, got.Metrics)
	assert.Empty(t, got.IPAddress)
}

func TestHTTPObserver_RetriesOnServerError(t *testing.T) {
	var calls int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt64(&calls, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	observer.NewHTTPObserver(srv.URL).Notify(context.Background(), "Alloc")

	assert.EqualValues(t, 2, atomic.LoadInt64(&calls), "первый 500 должен привести к ретраю")
}

func TestHTTPObserver_ClientErrorIsNotRetried(t *testing.T) {
	var calls int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	observer.NewHTTPObserver(srv.URL).Notify(context.Background(), "Alloc")

	assert.EqualValues(t, 1, atomic.LoadInt64(&calls))
}

func TestHTTPObserver_UnreachableServerDoesNotPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	obs := observer.NewHTTPObserver(url)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.NotPanics(t, func() { obs.Notify(ctx, "Alloc") })
}

func TestHTTPObserver_BrokenURL(t *testing.T) {
	obs := observer.NewHTTPObserver("://broken")
	assert.NotPanics(t, func() { obs.Notify(context.Background(), "Alloc") })
}

func TestNewObservers(t *testing.T) {
	dir := t.TempDir()
	auditFile := filepath.Join(dir, "audit.log")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	t.Run("оба бэкенда", func(t *testing.T) {
		obs, err := observer.NewObservers(auditFile, srv.URL)
		require.NoError(t, err)
		require.Len(t, obs, 2)

		obs.Notify(observer.WithClientIP(context.Background(), "1.2.3.4"), "Alloc")
		obs.NotifyBatch(context.Background(), []string{"A", "B"})
		require.NoError(t, obs.Close())

		data, err := os.ReadFile(auditFile)
		require.NoError(t, err)
		assert.Contains(t, string(data), "1.2.3.4")
		assert.Contains(t, string(data), `"metrics":["A","B"]`)
	})

	t.Run("без настроек", func(t *testing.T) {
		obs, err := observer.NewObservers("", "")
		require.NoError(t, err)
		assert.Empty(t, obs)
		assert.NoError(t, obs.Close())
	})

	t.Run("только URL", func(t *testing.T) {
		obs, err := observer.NewObservers("", srv.URL)
		require.NoError(t, err)
		assert.Len(t, obs, 1)
	})

	t.Run("недоступный путь к файлу", func(t *testing.T) {
		_, err := observer.NewObservers(filepath.Join(dir, "nope", "audit.log"), "")
		assert.Error(t, err)
	})
}

func TestObservers_CloseCollectsErrors(t *testing.T) {
	fileObs, err := observer.NewFileObserver(filepath.Join(t.TempDir(), "audit.log"))
	require.NoError(t, err)
	require.NoError(t, fileObs.Close())

	obs := observer.Observers{fileObs}
	assert.Error(t, obs.Close(), "повторное закрытие файла должно вернуть ошибку")
}

func TestFileObserver_WriteAfterCloseDoesNotPanic(t *testing.T) {
	fileObs, err := observer.NewFileObserver(filepath.Join(t.TempDir(), "audit.log"))
	require.NoError(t, err)
	require.NoError(t, fileObs.Close())

	assert.NotPanics(t, func() { fileObs.Notify(context.Background(), "Alloc") })
}

func TestClientIP_MissingValue(t *testing.T) {
	assert.Empty(t, observer.ClientIP(context.Background()))
	//nolint:staticcheck // проверяем, что чужой ключ того же вида не подхватывается
	assert.Empty(t, observer.ClientIP(context.WithValue(context.Background(), "client_ip", 42)))
}

func TestAuditEntry_Marshal(t *testing.T) {
	entry := observer.AuditEntry{TS: 100, Metrics: []string{"Alloc"}, IPAddress: "1.1.1.1"}

	data, err := entry.Marshal()
	require.NoError(t, err)
	assert.JSONEq(t, `{"ts":100,"metrics":["Alloc"],"ip_address":"1.1.1.1"}`, string(data))
}
