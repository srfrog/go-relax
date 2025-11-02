package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/srfrog/go-relax"
)

func TestCORSFiltersAreIsolated(t *testing.T) {
	const (
		allowedOrigin = "http://allowed.example.com"
		uniqueOrigin  = "http://unique.example.com"
	)

	requestWithOrigin := func(filter *Filter, origin string) *httptest.ResponseRecorder {
		svc := relax.NewService("/v1")
		svc.Use(filter)

		req := httptest.NewRequest(http.MethodGet, "/v1/", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		svc.ServeHTTP(rec, req)
		return rec
	}

	recAllowed := requestWithOrigin(&Filter{
		AllowOrigin: []string{allowedOrigin},
		Strict:      true,
	}, allowedOrigin)

	if recAllowed.Code != http.StatusOK {
		t.Fatalf("expected status 200 for allowed origin, got %d", recAllowed.Code)
	}
	if got := recAllowed.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
		t.Fatalf("expected Access-Control-Allow-Origin header %q, got %q", allowedOrigin, got)
	}

	recIsolated := requestWithOrigin(&Filter{
		AllowOrigin: []string{uniqueOrigin},
		Strict:      true,
	}, allowedOrigin)

	if recIsolated.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for unrelated origin on isolated filter, got %d", recIsolated.Code)
	}
}
