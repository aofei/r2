package r2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPathParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got, want := PathParam(req, "foo"), ""; got != want {
		t.Errorf("got %s, want %s", got, want)
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		pathParamsContextKey,
		"invalid",
	))
	if got, want := PathParam(req, "foo"), ""; got != want {
		t.Errorf("got %s, want %s", got, want)
	}

	pps := &pathParams{
		names:  []string{"foo", "bar"},
		values: []string{"bar", "foo"},
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		pathParamsContextKey,
		pps,
	))
	if got, want := PathParam(req, "foo"), "bar"; got != want {
		t.Errorf("got %s, want %s", got, want)
	} else if got, want := PathParam(req, "bar"), "foo"; got != want {
		t.Errorf("got %s, want %s", got, want)
	} else if got, want := PathParam(req, "foobar"), ""; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestPathParamNames(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if ppns := PathParamNames(req); ppns != nil {
		t.Errorf("got %v, want nil", ppns)
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		pathParamsContextKey,
		"invalid",
	))
	if ppns := PathParamNames(req); ppns != nil {
		t.Errorf("got %v, want nil", ppns)
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		pathParamsContextKey,
		&pathParams{
			names:  []string{"foo"},
			values: []string{"bar"},
		},
	))
	if ppns := PathParamNames(req); ppns == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(ppns), 1; got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestParamValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if ppvs := PathParamValues(req); ppvs != nil {
		t.Errorf("got %v, want nil", ppvs)
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		pathParamsContextKey,
		"invalid",
	))
	if ppvs := PathParamValues(req); ppvs != nil {
		t.Errorf("got %v, want nil", ppvs)
	}

	pps := &pathParams{
		names:  []string{"foo"},
		values: []string{"bar1", "bar2"},
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		pathParamsContextKey,
		pps,
	))
	if ppvs := PathParamValues(req); ppvs == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(ppvs), 1; got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}
