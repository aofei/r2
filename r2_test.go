package r2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestData(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if d := data(req); d != nil {
		t.Errorf("got %v, want nil", d)
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		dataKey,
		map[interface{}]interface{}{
			"foo": "bar",
		},
	))
	if d := data(req); d == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(d), 1; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if got, want := d["foo"], "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPathParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if pps := PathParams(req); pps != nil {
		t.Errorf("got %v, want nil", pps)
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{},
		},
	))
	if pps := PathParams(req); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(pps), 0; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: []*pathParamSlot{
					{number: 2, name: "first"},
					{number: 3, name: "second"},
				},
			},
			pathKey: "/base/foo/bar/foobar",
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
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: []*pathParamSlot{
					{number: 1, name: "first"},
					{number: 2, name: "second"},
				},
			},
			pathKey: "/foo/bar",
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
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: []*pathParamSlot{
					{number: 1, name: "first"},
					{number: 2, name: "second"},
				},
			},
			pathKey: "/foo//bar",
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
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: []*pathParamSlot{
					{number: 1, name: "first"},
					{number: 2, name: "second"},
				},
			},
			pathKey: "/foo/bar/",
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
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: []*pathParamSlot{
					{number: 1, name: "first"},
					{number: 2, name: "second"},
				},
			},
			pathKey: "/foo/",
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
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: []*pathParamSlot{
					{number: 0, name: "first"},
				},
			},
			pathKey: "/foo",
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
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: []*pathParamSlot{
					{number: 2, name: "second"},
				},
			},
			pathKey: "/foo",
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
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: []*pathParamSlot{
					{number: 2, name: "first"},
					{number: 3, name: "second"},
					{number: 4, name: "*"},
				},
			},
			pathKey: "/base/foo/bar/wildcard/foobar",
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
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: []*pathParamSlot{
					{number: 1, name: "first"},
					{number: 2, name: "second"},
					{number: 3, name: "*"},
				},
			},
			pathKey: "/foo/bar/wildcard/foobar",
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
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: []*pathParamSlot{
					{number: 1, name: "first"},
					{number: 2, name: "second"},
					{number: 3, name: "*"},
				},
			},
			pathKey: "/foo//bar///wildcard///foobar",
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
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: []*pathParamSlot{
					{number: 1, name: "first"},
					{number: 2, name: "second"},
					{number: 3, name: "*"},
				},
			},
			pathKey: "/foo/bar/",
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

func TestUnescapePathParamValue(t *testing.T) {
	if got, want := unescapePathParamValue("foo"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if got, want := unescapePathParamValue("foo%2F"), "foo/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if got, want := unescapePathParamValue("foo%%"), "foo%%"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
