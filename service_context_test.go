package relax

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type contextCapture struct {
	value interface{}
}

func (c *contextCapture) Index(ctx *Context) {
	c.value = ctx.Context.Value(contextKey{})
	ctx.Respond(map[string]string{"ok": "true"})
}

type contextKey struct{}

func TestAdapterPreservesRequestContext(t *testing.T) {
	resource := &contextCapture{}
	svc := NewService("/v1")
	svc.Resource(resource)

	req := httptest.NewRequest(http.MethodGet, "/v1/contextcapture/", nil)
	ctx := context.WithValue(req.Context(), contextKey{}, "retained")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	svc.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if resource.value != "retained" {
		t.Fatalf("expected request context value to propagate, got %#v", resource.value)
	}
}
