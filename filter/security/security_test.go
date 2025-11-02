package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/srfrog/go-relax"
)

type mockResource struct{}

func (m *mockResource) Index(ctx *relax.Context) {
	ctx.Respond(map[string]string{"ok": "true"})
}

func TestSecurityFilterLeavesCacheHeadersUntouchedByDefault(t *testing.T) {
	svc := relax.NewService("/v1")
	svc.Use(&Filter{})
	svc.Resource(&mockResource{})

	req := httptest.NewRequest(http.MethodGet, "/v1/mockresource/", nil)
	req.Header.Set("User-Agent", "test-suite")
	rec := httptest.NewRecorder()

	svc.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "" {
		t.Fatalf("expected Cache-Control header to be empty by default, got %q", got)
	}
	if got := rec.Header().Get("Pragma"); got != "" {
		t.Fatalf("expected Pragma header to be empty by default, got %q", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options header to be set to nosniff, got %q", got)
	}
}

func TestSecurityFilterAppliesCustomCachePolicy(t *testing.T) {
	const cachePolicy = "public, max-age=60"

	svc := relax.NewService("/v1")
	svc.Use(&Filter{CacheOptions: cachePolicy})
	svc.Resource(&mockResource{})

	req := httptest.NewRequest(http.MethodGet, "/v1/mockresource/", nil)
	req.Header.Set("User-Agent", "test-suite")
	rec := httptest.NewRecorder()

	svc.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != cachePolicy {
		t.Fatalf("expected Cache-Control header %q, got %q", cachePolicy, got)
	}
	if got := rec.Header().Get("Pragma"); got != securityPragmaDefault {
		t.Fatalf("expected Pragma header %q, got %q", securityPragmaDefault, got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options header to be set to nosniff, got %q", got)
	}
}
