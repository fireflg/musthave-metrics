package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/handler"
	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/observer"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/memory"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	repo := memory.NewMemoryRepository()
	svc := service.NewMetricsService(repo, nil)
	h := handler.NewMetricsHandler(svc, zaptest.NewLogger(t).Sugar(), "", nil, nil)
	return h.ServerRouter()
}

func doRequest(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestUpdateMetric_Plain(t *testing.T) {
	r := newTestRouter(t)

	tests := []struct {
		name     string
		path     string
		wantCode int
	}{
		{"gauge ok", "/update/gauge/Alloc/12.5", http.StatusOK},
		{"counter ok", "/update/counter/PollCount/7", http.StatusOK},
		{"unknown type", "/update/histogram/Alloc/1", http.StatusBadRequest},
		{"bad gauge value", "/update/gauge/Alloc/abc", http.StatusBadRequest},
		{"bad counter value", "/update/counter/PollCount/1.5", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := doRequest(t, r, http.MethodPost, tt.path, "")
			assert.Equal(t, tt.wantCode, rr.Code)
		})
	}
}

func TestGetMetric_Plain(t *testing.T) {
	r := newTestRouter(t)

	require.Equal(t, http.StatusOK, doRequest(t, r, http.MethodPost, "/update/gauge/Alloc/12.5", "").Code)
	require.Equal(t, http.StatusOK, doRequest(t, r, http.MethodPost, "/update/counter/PollCount/7", "").Code)

	rr := doRequest(t, r, http.MethodGet, "/value/gauge/Alloc", "")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "12.5", rr.Body.String())

	rr = doRequest(t, r, http.MethodGet, "/value/counter/PollCount", "")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "7", rr.Body.String())

	rr = doRequest(t, r, http.MethodGet, "/value/gauge/Missing", "")
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetMetric_UnknownTypeReturns404(t *testing.T) {
	r := newTestRouter(t)

	require.Equal(t, http.StatusOK, doRequest(t, r, http.MethodPost, "/update/gauge/Alloc/1", "").Code)

	rr := doRequest(t, r, http.MethodGet, "/value/histogram/Alloc", "")
	assert.NotEqual(t, http.StatusOK, rr.Code)
}

func TestUpdateMetricJSON(t *testing.T) {
	r := newTestRouter(t)

	rr := doRequest(t, r, http.MethodPost, "/update/", `{"id":"Alloc","type":"gauge","value":3.5}`)
	require.Equal(t, http.StatusOK, rr.Code)

	rr = doRequest(t, r, http.MethodPost, "/value/", `{"id":"Alloc","type":"gauge"}`)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, 3.5, resp["value"])
}

func TestUpdateMetricJSON_BrokenBody(t *testing.T) {
	r := newTestRouter(t)

	rr := doRequest(t, r, http.MethodPost, "/update/", `{"id":`)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.NotContains(t, rr.Body.String(), `"status":"ok"`)
}

func TestUpdateMetricJSON_ServiceError(t *testing.T) {
	r := newTestRouter(t)

	rr := doRequest(t, r, http.MethodPost, "/update/", `{"id":"Alloc","type":"gauge"}`)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), `"status":"ok"`)
}

func TestUpdateMetricJSONBatch(t *testing.T) {
	r := newTestRouter(t)

	body := `[{"id":"Alloc","type":"gauge","value":1.5},{"id":"PollCount","type":"counter","delta":3}]`
	rr := doRequest(t, r, http.MethodPost, "/updates/", body)
	require.Equal(t, http.StatusOK, rr.Code)

	rr = doRequest(t, r, http.MethodGet, "/value/counter/PollCount", "")
	assert.Equal(t, "3", rr.Body.String())
}

func TestUpdateMetricJSONBatch_BrokenBody(t *testing.T) {
	r := newTestRouter(t)

	rr := doRequest(t, r, http.MethodPost, "/updates/", `[{"id":`)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.NotContains(t, rr.Body.String(), `"status":"ok"`)
}

func TestUpdateMetricJSONBatch_ServiceError(t *testing.T) {
	r := newTestRouter(t)

	rr := doRequest(t, r, http.MethodPost, "/updates/", `[{"id":"PollCount","type":"counter"}]`)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), `"status":"ok"`)
}

func TestGetMetricJSON_NotFound(t *testing.T) {
	r := newTestRouter(t)

	rr := doRequest(t, r, http.MethodPost, "/value/", `{"id":"Missing","type":"gauge"}`)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetMetricJSON_BrokenBody(t *testing.T) {
	r := newTestRouter(t)

	rr := doRequest(t, r, http.MethodPost, "/value/", `not json`)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCheckDBAndRoot(t *testing.T) {
	r := newTestRouter(t)

	assert.Equal(t, http.StatusOK, doRequest(t, r, http.MethodGet, "/ping", "").Code)
	assert.Equal(t, http.StatusOK, doRequest(t, r, http.MethodGet, "/", "").Code)
}

type recordingObserver struct {
	ips []string
}

func (o *recordingObserver) Notify(ctx context.Context, _ string) {
	o.ips = append(o.ips, observer.ClientIP(ctx))
}

func (o *recordingObserver) NotifyBatch(ctx context.Context, _ []string) {
	o.ips = append(o.ips, observer.ClientIP(ctx))
}

func TestAuditReceivesClientIP(t *testing.T) {
	obs := &recordingObserver{}
	svc := service.NewMetricsService(memory.NewMemoryRepository(), observer.Observers{obs})
	h := handler.NewMetricsHandler(svc, zaptest.NewLogger(t).Sugar(), "", nil, nil)
	r := h.ServerRouter()

	req := httptest.NewRequest(http.MethodPost, "/update/",
		bytes.NewBufferString(`{"id":"Alloc","type":"gauge","value":1}`))
	req.Header.Set("X-Real-IP", "10.1.2.3")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.NotEmpty(t, obs.ips)
	assert.Equal(t, "10.1.2.3", obs.ips[0])
}

func TestAuditFallsBackToRemoteAddr(t *testing.T) {
	obs := &recordingObserver{}
	svc := service.NewMetricsService(memory.NewMemoryRepository(), observer.Observers{obs})
	h := handler.NewMetricsHandler(svc, zaptest.NewLogger(t).Sugar(), "", nil, nil)
	r := h.ServerRouter()

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/1", nil)
	req.RemoteAddr = "192.0.2.10:54321"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.NotEmpty(t, obs.ips)
	assert.Equal(t, "192.0.2.10", obs.ips[0])
}

func TestSecretKeyIsWired(t *testing.T) {
	svc := service.NewMetricsService(memory.NewMemoryRepository(), nil)
	h := handler.NewMetricsHandler(svc, zaptest.NewLogger(t).Sugar(), "secret", nil, nil)
	r := h.ServerRouter()

	body := `{"id":"Alloc","type":"gauge","value":1}`

	req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HashSHA256", "deadbeef")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code, "запрос с неверной подписью должен быть отклонён")

	rr = doRequest(t, r, http.MethodPost, "/update/", body)
	assert.Equal(t, http.StatusOK, rr.Code, "клиент вправе не подписывать запрос")
}

func TestModelsConstantsUsedInRouter(t *testing.T) {
	r := newTestRouter(t)

	rr := doRequest(t, r, http.MethodPost, "/update/"+models.Gauge+"/Alloc/1", "")
	assert.Equal(t, http.StatusOK, rr.Code)
}
