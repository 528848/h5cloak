package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"cloak/internal/config"
	"cloak/internal/pkg/requestid"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	cfg := &config.Config{}
	cfg.App.Env = "test"
	return NewRouter(Deps{Cfg: cfg, Log: zap.NewNop(), Metrics: prometheus.NewRegistry()})
}

func TestVersionEndpoint(t *testing.T) {
	r := newTestRouter(t)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/version", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"].(float64) != 200 {
		t.Fatalf("code = %v", body["code"])
	}
	if rec.Header().Get(requestid.HeaderName) == "" {
		t.Fatal("missing request id header")
	}
}

func TestNoRouteReturnsRuoYiBody(t *testing.T) {
	r := newTestRouter(t)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/not-exists", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["code"].(float64) != 404 || body["msg"] == "" {
		t.Fatalf("unexpected body %s", rec.Body.String())
	}
}

func TestRequestIDIsPreserved(t *testing.T) {
	r := newTestRouter(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	req.Header.Set(requestid.HeaderName, "abc123")
	r.ServeHTTP(rec, req)
	if got := rec.Header().Get(requestid.HeaderName); got != "abc123" {
		t.Fatalf("request id = %q", got)
	}
}
