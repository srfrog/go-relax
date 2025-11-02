package relax

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type testResource struct{}

func (t *testResource) Index(ctx *Context) {
	ctx.Respond(map[string]string{"status": "ok"})
}

type trackingFilter struct {
	name  string
	calls *int
	order *[]string
}

func (f *trackingFilter) Run(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		if f.calls != nil {
			*f.calls++
		}
		if f.order != nil {
			*f.order = append(*f.order, f.name)
		}
		next(ctx)
	}
}

func TestResourceFiltersAreApplied(t *testing.T) {
	svc := NewService("/v1")

	var sequence []string
	resourceCalls := 0
	routeCalls := 0

	res := svc.Resource(&testResource{}, &trackingFilter{name: "resource", calls: &resourceCalls, order: &sequence})
	res.GET("items", func(ctx *Context) {
		ctx.Respond(map[string]string{"handled": "true"})
	}, &trackingFilter{name: "route", calls: &routeCalls, order: &sequence})

	req := httptest.NewRequest(http.MethodGet, res.Path(false)+"/items", nil)
	rec := httptest.NewRecorder()

	svc.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rec.Code)
	}
	if resourceCalls != 1 {
		t.Fatalf("expected resource filter to run once, ran %d times", resourceCalls)
	}
	if routeCalls != 1 {
		t.Fatalf("expected route filter to run once, ran %d times", routeCalls)
	}
	if len(sequence) != 2 {
		t.Fatalf("expected two filter executions, got %d (%v)", len(sequence), sequence)
	}
	if sequence[0] != "resource" || sequence[1] != "route" {
		t.Fatalf("expected filter execution order [resource route], got %v", sequence)
	}
}
