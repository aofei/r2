package r2

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterSub(t *testing.T) {
	r := &Router{}
	sr := r.Sub("/prefix", MiddlewareFunc(func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			next.ServeHTTP(rw, req)
		})
	}))
	if sr.Parent != r {
		t.Errorf("got %v, want %v", sr.Parent, r)
	} else if want := "/prefix"; sr.PathPrefix != want {
		t.Errorf("got %q, want %q", sr.PathPrefix, want)
	} else if got, want := len(sr.Middlewares), 1; got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestRouterHandle(t *testing.T) {
	r := &Router{}
	r.Handle("", "/", http.NotFoundHandler())
	if r.routeTree == nil {
		t.Fatal("unexpected nil")
	} else if r.routeTree.handlers == nil {
		t.Fatal("unexpected nil")
	} else if r.registeredRoutes == nil {
		t.Fatal("unexpected nil")
	} else if r.overridableRoutes == nil {
		t.Fatal("unexpected nil")
	}

	r = &Router{}
	sr := &Router{
		Parent:     r,
		PathPrefix: "/sub",
	}
	sr.Handle("", "/", http.NotFoundHandler(), nil)
	if sr.routeTree != nil {
		t.Errorf("got %v, want nil", sr.routeTree)
	} else if sr.registeredRoutes != nil {
		t.Errorf("got %v, want nil", sr.registeredRoutes)
	} else if sr.overridableRoutes != nil {
		t.Errorf("got %v, want nil", sr.overridableRoutes)
	} else if r.routeTree == nil {
		t.Fatal("unexpected nil")
	} else if r.routeTree.handlers == nil {
		t.Fatal("unexpected nil")
	} else if r.registeredRoutes == nil {
		t.Fatal("unexpected nil")
	} else if r.overridableRoutes == nil {
		t.Fatal("unexpected nil")
	}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		r = &Router{}
		r.Handle("", "", http.NotFoundHandler())
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		r = &Router{}
		r.Handle("", "foo", http.NotFoundHandler())
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		r = &Router{}
		r.Handle("", "/:foo:bar", http.NotFoundHandler())
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		r = &Router{}
		r.Handle("", "/foo/*/*", http.NotFoundHandler())
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		r = &Router{}
		r.Handle("", "/foo/*/bar", http.NotFoundHandler())
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		r = &Router{}
		r.Handle("", "/foo/:bar*", http.NotFoundHandler())
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		r = &Router{}
		r.Handle("", "/foo/:bar1", http.NotFoundHandler())
		r.Handle("", "/foo/:bar2", http.NotFoundHandler())
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		r = &Router{}
		r.Handle("", "/", nil)
	}()
}

func TestRouterHandler(t *testing.T) {
	h := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprint(rw, Values(req)["middleware"])
	})
	mwf := MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			Values(req)["middleware"] = true
			next.ServeHTTP(rw, req)
		})
	})

	r := &Router{}
	r.Handle("", "/", h, mwf)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "true"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	r = &Router{}
	sr := &Router{
		Parent:     r,
		PathPrefix: "/foo",
	}
	sr.Handle("", "/", h, mwf)
	req = httptest.NewRequest(http.MethodGet, "/foo/", nil)
	rec = httptest.NewRecorder()
	sr.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "true"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	r = &Router{}
	r.Handle("", "/", h, mwf)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RequestURI = ""
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusNotFound; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Not Found\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	r = &Router{}
	r.Handle("", "/", h, mwf)
	req = httptest.NewRequest(http.MethodGet, "/?foo=bar", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "true"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := req.URL.Query().Get("foo"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	h = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		Defer(req, func() {
			fmt.Fprint(rw, "deferred")
		})
	})

	r = &Router{}
	r.Handle("", "/", h)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "deferred"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}
}

func TestRouterHandler_static(t *testing.T) {
	r := &Router{}
	r.Handle(http.MethodGet, "/", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /")
	}))
	r.Handle(http.MethodGet, "/foobar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foobar")
	}))
	r.Handle(http.MethodGet, "/foo/bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foo/bar")
	}))
	r.Handle(http.MethodGet, "/foo/bar/", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foo/bar/")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "//", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar/", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/bar/"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusNotFound; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Not Found\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar/foobar", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusNotFound; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Not Found\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodHead, "/", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusMethodNotAllowed; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Method Not Allowed\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}
}

func TestRouterHandler_param(t *testing.T) {
	r := &Router{}

	var hReq *http.Request
	r.Handle(http.MethodGet, "/:foobar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /:foobar")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foobar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("foobar"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "//", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foobar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("foobar"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foobar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("foobar"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar/", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusNotFound; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Not Found\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	r.Handle(http.MethodGet, "/foo:bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foo:bar")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foo", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("bar"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("bar"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r.Handle(http.MethodGet, "/:foo/:bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /:foo/:bar")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foo/:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("foo"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("bar"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRouterHandler_wildcardParam(t *testing.T) {
	r := &Router{}

	var hReq *http.Request
	r.Handle(http.MethodGet, "/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /*")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "//", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar/", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "foobar/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar//", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "foobar//"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "foo/bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar/", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "foo/bar/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar//", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "foo/bar//"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r.Handle(http.MethodGet, "/foobar*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foobar*")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar/", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar//", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "//"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r.Handle(http.MethodGet, "/foobar/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foobar/*")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foobar/", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r.Handle(http.MethodGet, "/foobar2/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foobar2/*")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foobar2/", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar2/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r.Handle(http.MethodGet, "/foo/bar/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foo/bar/*")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foo/bar/", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/bar/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusMovedPermanently; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if loc, err := recr.Location(); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if got, want := loc.String(), "/foo/bar/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar?foo=bar", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusMovedPermanently; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if loc, err := recr.Location(); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if got, want := loc.String(), "/foo/bar/?foo=bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodHead, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusMovedPermanently; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if loc, err := recr.Location(); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if got, want := loc.String(), "/foo/bar/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodPost, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusMovedPermanently; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if loc, err := recr.Location(); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if got, want := loc.String(), "/foo/bar/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r.Handle("", "/foo/bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foo/bar")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodHead, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodPost, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	r.Handle(http.MethodGet, "/bar/foo*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /bar/foo*")
	}))

	req = httptest.NewRequest(http.MethodGet, "/bar/foo", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /bar/foo*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/bar/", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "bar/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRouterHandler_mixed(t *testing.T) {
	r := &Router{}

	var hReq *http.Request
	r.Handle(http.MethodGet, "/", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /")
	}))
	r.Handle(http.MethodGet, "/foo", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foo")
	}))
	r.Handle(http.MethodGet, "/bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /bar")
	}))
	r.Handle(http.MethodGet, "/foobar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foobar")
	}))
	r.Handle(http.MethodGet, "/:foobar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /:foobar")
	}))
	r.Handle(http.MethodGet, "/foo/:bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foo/:bar")
	}))
	r.Handle(http.MethodGet, "/foo:bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foo:bar")
	}))
	r.Handle(http.MethodGet, "/:foo/:bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /:foo/:bar")
	}))
	r.Handle(http.MethodGet, "/foobar*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foobar*")
	}))
	r.Handle(http.MethodGet, "/foobar/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foobar/*")
	}))
	r.Handle(http.MethodGet, "/foo/:bar/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foo/:bar/*")
	}))
	r.Handle(http.MethodGet, "/foo:bar/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /foo:bar/*")
	}))
	r.Handle(http.MethodGet, "/:foo/:bar/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /:foo/:bar/*")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/bar", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec = httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/barfoo", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foobar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("foobar"), "barfoo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("bar"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("bar"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/fooobar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("bar"), "obar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/bar/foo", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foo/:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("foo"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("bar"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobarfoobar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar/foobar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar/foobar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/:bar/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("bar"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foofoobar/foobar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo:bar/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("bar"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/bar/foo/foobar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foo/:bar/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("foo"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("bar"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRouterHandler_fallback(t *testing.T) {
	r := &Router{}

	var hReq *http.Request
	r.Handle(http.MethodGet, "/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /*")
	}))
	r.Handle(http.MethodGet, "/:foo/:bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hReq = req
		fmt.Fprint(rw, "GET /:foo/:bar")
	}))

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	rec := httptest.NewRecorder()
	r.Handler(req).ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foo/:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("foo"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := pps.Get("bar"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar/foobar", nil)
	rec = httptest.NewRecorder()
	hReq = nil
	r.Handler(req).ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if hReq == nil {
		t.Fatal("unexpected nil")
	} else if pps := PathParams(hReq); pps == nil {
		t.Fatal("unexpected nil")
	} else if got, want := pps.Get("*"), "foo/bar/foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRouterNotFoundHandler(t *testing.T) {
	r := &Router{}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.notFoundHandler().ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusNotFound; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Not Found\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	r = &Router{
		NotFoundHandler: http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			http.Error(rw, "custom", http.StatusNotFound)
		}),
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	r.notFoundHandler().ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusNotFound; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "custom\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}
}

func TestRouterMethodNotAllowedHandler(t *testing.T) {
	r := &Router{}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.methodNotAllowedHandler().ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusMethodNotAllowed; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Method Not Allowed\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	r = &Router{
		MethodNotAllowedHandler: http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			http.Error(rw, "custom", http.StatusMethodNotAllowed)
		}),
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	r.methodNotAllowedHandler().ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusMethodNotAllowed; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "custom\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}
}

func TestRouterTSRHandler(t *testing.T) {
	r := &Router{}

	req := httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec := httptest.NewRecorder()
	r.tsrHandler().ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusMovedPermanently; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if loc, err := recr.Location(); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if got, want := loc.String(), "/foobar/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RequestURI = ""
	rec = httptest.NewRecorder()
	r.tsrHandler().ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusMovedPermanently; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if loc, err := recr.Location(); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if got, want := loc.String(), "/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RequestURI = "?foo=bar"
	rec = httptest.NewRecorder()
	r.tsrHandler().ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusMovedPermanently; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if loc, err := recr.Location(); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if got, want := loc.String(), "/?foo=bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r = &Router{
		TSRHandler: http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			http.Error(
				rw,
				http.StatusText(http.StatusNotFound),
				http.StatusNotFound,
			)
		}),
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	r.tsrHandler().ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusNotFound; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Not Found\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}
}

func TestRouterServeHTTP(t *testing.T) {
	r1 := &Router{
		NotFoundHandler: http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			http.Error(rw, "r1: not found", http.StatusNotFound)
		}),
		routeTree: &routeNode{
			handlers: map[string]http.Handler{},
		},
		registeredRoutes:  map[string]bool{},
		overridableRoutes: map[string]bool{},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r1.ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusNotFound; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "r1: not found\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	r2 := &Router{
		Parent: r1,
		NotFoundHandler: http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			http.Error(rw, "r2: not found", http.StatusNotFound)
		}),
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	r2.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusNotFound; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "r1: not found\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}
}

func TestRouteNodeChild(t *testing.T) {
	rn := &routeNode{
		children: []*routeNode{{
			label: 'a',
			typ:   staticRouteNode,
		}},
	}

	if crn := rn.child('b', paramRouteNode); crn != nil {
		t.Errorf("got %v, want nil", crn)
	}

	if crn := rn.child('a', staticRouteNode); crn == nil {
		t.Fatal("unexpected nil")
	}
}

func TestRouteNodeLChild(t *testing.T) {
	rn := &routeNode{
		children: []*routeNode{{
			label: 'a',
		}},
	}

	if crn := rn.lChild('b'); crn != nil {
		t.Errorf("got %v, want nil", crn)
	}

	if crn := rn.lChild('a'); crn == nil {
		t.Fatal("unexpected nil")
	}
}

func TestRouteNodeTChild(t *testing.T) {
	rn := &routeNode{
		children: []*routeNode{{
			typ: staticRouteNode,
		}},
	}

	if crn := rn.tChild(paramRouteNode); crn != nil {
		t.Errorf("got %v, want nil", crn)
	}

	if crn := rn.tChild(staticRouteNode); crn == nil {
		t.Fatal("unexpected nil")
	}
}
