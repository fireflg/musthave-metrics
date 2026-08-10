package middleware_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/middleware"
	"go.uber.org/zap/zaptest"
)

func mustParseCIDR(t *testing.T, cidr string) *net.IPNet {
	t.Helper()
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("failed to parse CIDR: %v", err)
	}
	return ipNet
}

func TestTrustedSubnetMiddleware_IPInSubnet(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	subnet := mustParseCIDR(t, "192.168.1.0/24")

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Real-IP", "192.168.1.42")
	rr := httptest.NewRecorder()

	called := false
	h := middleware.TrustedSubnetMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}, subnet, logger)

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestTrustedSubnetMiddleware_IPOutsideSubnet(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	subnet := mustParseCIDR(t, "192.168.1.0/24")

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")
	rr := httptest.NewRecorder()

	called := false
	h := middleware.TrustedSubnetMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}, subnet, logger)

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", rr.Code)
	}
	if called {
		t.Fatal("expected handler not to be called")
	}
}

func TestTrustedSubnetMiddleware_MissingHeader(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	subnet := mustParseCIDR(t, "192.168.1.0/24")

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()

	h := middleware.TrustedSubnetMiddleware(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("expected handler not to be called")
	}, subnet, logger)

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", rr.Code)
	}
}
