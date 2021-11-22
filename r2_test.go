package r2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContext(t *testing.T) {
	ctx := Context()
	if ctx == nil {
		t.Fatal("unexpected nil")
	} else if dc, ok := ctx.(*dataContext); !ok {
		t.Error("want true")
	} else if dc.d == nil {
		t.Fatal("unexpected nil")
	}
}

func TestPathParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got, want := PathParam(req, "foo"), ""; got != want {
		t.Errorf("got %s, want %s", got, want)
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		dataContextKey,
		"invalid",
	))
	if got, want := PathParam(req, "foo"), ""; got != want {
		t.Errorf("got %s, want %s", got, want)
	}

	d := &data{
		pathParamNames:  []string{"foo", "bar"},
		pathParamValues: []string{"bar", "foo"},
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		dataContextKey,
		d,
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
		dataContextKey,
		"invalid",
	))
	if ppns := PathParamNames(req); ppns != nil {
		t.Errorf("got %v, want nil", ppns)
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		dataContextKey,
		&data{
			pathParamNames:  []string{"foo"},
			pathParamValues: []string{"bar"},
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
		dataContextKey,
		"invalid",
	))
	if ppvs := PathParamValues(req); ppvs != nil {
		t.Errorf("got %v, want nil", ppvs)
	}

	d := &data{
		pathParamNames:  []string{"foo"},
		pathParamValues: []string{"bar1", "bar2"},
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		dataContextKey,
		d,
	))
	if ppvs := PathParamValues(req); ppvs == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(ppvs), 1; got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestDataContext(t *testing.T) {
	dc := &dataContext{d: &data{}}
	if deadline, ok := dc.Deadline(); !deadline.IsZero() {
		t.Error("want true")
	} else if ok {
		t.Error("want false")
	} else if got := dc.Done(); got != nil {
		t.Errorf("got %v, want nil", got)
	} else if got := dc.Err(); got != nil {
		t.Errorf("got %v, want nil", got)
	} else if dc.Value(dataContextKey) == nil {
		t.Fatal("unexpected nil")
	} else if got := dc.Value(contextKey(255)); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}
