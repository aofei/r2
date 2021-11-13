package r2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestPathParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 0; got != want {
		t.Errorf("got %d, want %d", got, want)
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		matchDataContextKey,
		"invalid",
	))
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 0; got != want {
		t.Errorf("got %d, want %d", got, want)
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		matchDataContextKey,
		&matchData{
			path: "/base/foo/bar/foobar",
			pathParamSlots: []*pathParamSlot{
				{number: 2, name: "first"},
				{number: 3, name: "second"},
			},
		},
	))
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 2; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if got, want := pps.Get("first"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("second"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		matchDataContextKey,
		&matchData{
			path: "/foo/bar",
			pathParamSlots: []*pathParamSlot{
				{number: 1, name: "first"},
				{number: 2, name: "second"},
			},
		},
	))
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 2; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if got, want := pps.Get("first"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("second"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		matchDataContextKey,
		&matchData{
			path: "/foo//bar",
			pathParamSlots: []*pathParamSlot{
				{number: 1, name: "first"},
				{number: 2, name: "second"},
			},
		},
	))
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 2; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if got, want := pps.Get("first"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("second"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		matchDataContextKey,
		&matchData{
			path: "/foo/bar/",
			pathParamSlots: []*pathParamSlot{
				{number: 1, name: "first"},
				{number: 2, name: "second"},
			},
		},
	))
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 2; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if got, want := pps.Get("first"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("second"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		matchDataContextKey,
		&matchData{
			path: "/foo/",
			pathParamSlots: []*pathParamSlot{
				{number: 1, name: "first"},
				{number: 2, name: "second"},
			},
		},
	))
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 2; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if got, want := pps.Get("first"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("second"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if _, ok := pps["second"]; !ok {
		t.Error("want true")
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		matchDataContextKey,
		&matchData{
			path: "/foo",
			pathParamSlots: []*pathParamSlot{
				{number: 0, name: "first"},
			},
		},
	))
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 0; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if got, want := pps.Get("first"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if _, ok := pps["first"]; ok {
		t.Error("want false")
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		matchDataContextKey,
		&matchData{
			path: "/foo",
			pathParamSlots: []*pathParamSlot{
				{number: 2, name: "second"},
			},
		},
	))
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 0; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if got, want := pps.Get("first"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if _, ok := pps["first"]; ok {
		t.Error("want false")
	} else if got, want := pps.Get("second"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if _, ok := pps["second"]; ok {
		t.Error("want false")
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		matchDataContextKey,
		&matchData{
			path: "/base/foo/bar/wildcard/foobar",
			pathParamSlots: []*pathParamSlot{
				{number: 2, name: "first"},
				{number: 3, name: "second"},
				{number: 4, name: "*"},
			},
		},
	))
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 3; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if got, want := pps.Get("first"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("second"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("*"), "wildcard/foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		matchDataContextKey,
		&matchData{
			path: "/foo/bar/wildcard/foobar",
			pathParamSlots: []*pathParamSlot{
				{number: 1, name: "first"},
				{number: 2, name: "second"},
				{number: 3, name: "*"},
			},
		},
	))
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 3; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if got, want := pps.Get("first"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("second"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("*"), "wildcard/foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		matchDataContextKey,
		&matchData{
			path: "/foo//bar///wildcard///foobar",
			pathParamSlots: []*pathParamSlot{
				{number: 1, name: "first"},
				{number: 2, name: "second"},
				{number: 3, name: "*"},
			},
		},
	))
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 3; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if got, want := pps.Get("first"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("second"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("*"), "wildcard///foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		matchDataContextKey,
		&matchData{
			path: "/foo/bar/",
			pathParamSlots: []*pathParamSlot{
				{number: 1, name: "first"},
				{number: 2, name: "second"},
				{number: 3, name: "*"},
			},
		},
	))
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 3; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if got, want := pps.Get("first"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("second"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if _, ok := pps["*"]; !ok {
		t.Error("want true")
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}
}

func TestAddPathParam(t *testing.T) {
	vs := url.Values{}
	addPathParam(vs, "foo1", "bar1")
	addPathParam(vs, "foo1", "bar2")
	addPathParam(vs, "foo2", "bar%2F")
	addPathParam(vs, "foo3", "bar%%")

	if got, want := vs.Get("foo1"), "bar1"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if got, want := len(vs["foo1"]), 2; got != want {
		t.Errorf("got %d, want %d", got, want)
	}

	if got, want := vs.Get("foo2"), "bar/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if got, want := vs.Get("foo3"), "bar%%"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
