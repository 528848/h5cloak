package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"cloak/internal/config"
	"cloak/internal/pkg/requestid"
)

func TestHealth(t *testing.T) {
	h := New(Deps{Cfg: &config.Config{}, Log: zap.NewNop(), Metrics: prometheus.NewRegistry()})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_cloak/health", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("unexpected %d %q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get(requestid.HeaderName) == "" {
		t.Fatal("missing request id")
	}
}

func TestUnmatchedIs404WithoutServerHeader(t *testing.T) {
	h := New(Deps{Cfg: &config.Config{}, Log: zap.NewNop(), Metrics: prometheus.NewRegistry()})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/anything", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("Server") != "" {
		t.Fatal("server header must not be exposed")
	}
}
