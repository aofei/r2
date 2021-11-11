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
				pathParamSlots: map[int]*pathParamSlot{
					2: {name: "first"},
					3: {name: "second"},
				},
				maxPathParamSlot: 3,
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
				pathParamSlots: map[int]*pathParamSlot{
					1: {name: "first"},
					2: {name: "second"},
				},
				maxPathParamSlot: 2,
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
				pathParamSlots: map[int]*pathParamSlot{
					1: {name: "first"},
					2: {name: "second"},
				},
				maxPathParamSlot: 2,
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
				pathParamSlots: map[int]*pathParamSlot{
					1: {name: "first"},
					2: {name: "second"},
				},
				maxPathParamSlot: 2,
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
				pathParamSlots: map[int]*pathParamSlot{
					1: {name: "first"},
					2: {name: "second"},
				},
				maxPathParamSlot: 2,
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
	} else if !pps.Has("second") {
		t.Error("want true")
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: map[int]*pathParamSlot{
					1: {name: "first"},
				},
				maxPathParamSlot: 0,
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
	} else if pps.Has("first") {
		t.Error("want false")
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: map[int]*pathParamSlot{
					2: {name: "second"},
				},
				maxPathParamSlot: 2,
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
	} else if pps.Has("first") {
		t.Error("want false")
	} else if got, want := pps.Get("second"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if pps.Has("second") {
		t.Error("want false")
	} else if !reflect.DeepEqual(PathParams(req), pps) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		dataKey,
		map[interface{}]interface{}{
			routeNodeKey: &routeNode{
				pathParamSlots: map[int]*pathParamSlot{
					2: {name: "first"},
					3: {name: "second"},
					4: {name: "*"},
				},
				maxPathParamSlot: 4,
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
				pathParamSlots: map[int]*pathParamSlot{
					1: {name: "first"},
					2: {name: "second"},
					3: {name: "*"},
				},
				maxPathParamSlot: 3,
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
				pathParamSlots: map[int]*pathParamSlot{
					1: {name: "first"},
					2: {name: "second"},
					3: {name: "*"},
				},
				maxPathParamSlot: 3,
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
				pathParamSlots: map[int]*pathParamSlot{
					1: {name: "first"},
					2: {name: "second"},
					3: {name: "*"},
				},
				maxPathParamSlot: 3,
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
	} else if !pps.Has("*") {
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

func TestValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if vs := Values(req); vs != nil {
		t.Errorf("got %v, want nil", vs)
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		dataKey,
		map[interface{}]interface{}{},
	))
	if vs := Values(req); vs == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(vs), 0; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if !reflect.DeepEqual(Values(req), vs) {
		t.Error("want true")
	}

	req = req.WithContext(context.WithValue(
		context.Background(),
		dataKey,
		map[interface{}]interface{}{
			valuesKey: map[interface{}]interface{}{
				"foo": "bar",
			},
		},
	))
	if vs := Values(req); vs == nil {
		t.Fatal("unexpected nil")
	} else if got, want := len(vs), 1; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if got, want := vs["foo"], "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDefer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	Defer(req, func() {})
	if di := context.Background().Value(dataKey); di != nil {
		t.Errorf("got %v, want nil", di)
	}

	d := map[interface{}]interface{}{}
	req = req.WithContext(context.WithValue(
		context.Background(),
		dataKey,
		d,
	))

	Defer(req, nil)
	if _, ok := d[deferredFuncsKey]; ok {
		t.Error("want false")
	}

	Defer(req, func() {})
	Defer(req, func() {})
	if dfsi, ok := d[deferredFuncsKey]; !ok {
		t.Error("want true")
	} else if got, want := len(dfsi.([]func())), 2; got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}
