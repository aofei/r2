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
	} else if r.routeTree.methodHandlerSet == nil {
		t.Fatal("unexpected nil")
	} else if r.registeredRoutes == nil {
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
	} else if r.routeTree == nil {
		t.Fatal("unexpected nil")
	} else if r.routeTree.methodHandlerSet == nil {
		t.Fatal("unexpected nil")
	} else if r.registeredRoutes == nil {
		t.Fatal("unexpected nil")
	}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		r = &Router{}
		r.Handle("_", "/", http.NotFoundHandler())
	}()

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
		r.Handle("", "/foo/:", http.NotFoundHandler())
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		r = &Router{}
		r.Handle("", "/foo/:bar/:bar", http.NotFoundHandler())
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
		fmt.Fprint(rw, req.Host == "www.example.com")
	})
	mf := MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			req.Host = "www.example.com"
			next.ServeHTTP(rw, req)
		})
	})

	r := &Router{}
	r.Handle("", "/", h, mf)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mh, req := r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "true"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	r = &Router{}
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusNotFound; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Not Found\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	r = &Router{}
	sr := &Router{
		Parent:     r,
		PathPrefix: "/foo",
	}
	sr.Handle("", "/", h, mf)
	req = httptest.NewRequest(http.MethodGet, "/foo/", nil)
	rec = httptest.NewRecorder()
	mh, req = sr.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "true"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	r = &Router{}
	r.Handle("", "/", h, mf)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RequestURI = ""
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusNotFound; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Not Found\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	r = &Router{}
	r.Handle("", "/", h, mf)
	req = httptest.NewRequest(http.MethodGet, "/?foo=bar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
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
}

func TestRouterHandler_static(t *testing.T) {
	r := &Router{}
	r.Handle(http.MethodGet, "/", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /")
	}))
	r.Handle("custom", "/foo", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "custom /foo")
	}))
	r.Handle("", "/foobar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "_ /foobar")
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
	mh, req := r.Handler(req)
	mh.ServeHTTP(rec, req)
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
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest("custom", "/foo", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "custom /foo"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest("bar", "/foo", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusMethodNotAllowed; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Method Not Allowed\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	for _, method := range []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	} {
		req = httptest.NewRequest(method, "/foobar", nil)
		rec = httptest.NewRecorder()
		mh, req = r.Handler(req)
		mh.ServeHTTP(rec, req)
		recr = rec.Result()
		if want := http.StatusOK; recr.StatusCode != want {
			t.Errorf("got %d, want %d", recr.StatusCode, want)
		} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
			t.Fatalf("unexpected error %q", err)
		} else if want := "_ /foobar"; string(b) != want {
			t.Errorf("got %q, want %q", b, want)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
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
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/bar/"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/bar/foo", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusNotFound; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Not Found\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}
}

func TestRouterHandler_param(t *testing.T) {
	r := &Router{}

	r.Handle(http.MethodGet, "/:foobar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /:foobar")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mh, req := r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foobar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "foobar"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "//", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foobar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "foobar"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodHead, "/", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusMethodNotAllowed; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Method Not Allowed\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foobar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "foobar"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar/", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
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
		fmt.Fprint(rw, "GET /foo:bar")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foo", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "bar"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "bar"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r.Handle(http.MethodGet, "/:foo/:bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /:foo/:bar")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	req = req.WithContext(&dataContext{d: &data{}})
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foo/:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "foo"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := PathParam(req, "bar"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRouterHandler_wildcardParam(t *testing.T) {
	r := &Router{}

	r.Handle(http.MethodGet, "/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /*")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mh, req := r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "//", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodHead, "/", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusMethodNotAllowed; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Method Not Allowed\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar/", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "foobar/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar//", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "foobar//"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "foo/bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar/", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "foo/bar/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar//", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "foo/bar//"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r.Handle(http.MethodGet, "/foobar*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foobar*")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar/", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar//", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "//"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r.Handle(http.MethodGet, "/foobar/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foobar/*")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foobar/", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r.Handle(http.MethodGet, "/foobar2/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foobar2/*")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foobar2/", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar2/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r.Handle(http.MethodGet, "/foo/bar/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foo/bar/*")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foo/bar/", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/bar/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
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
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
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
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusMovedPermanently; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if loc, err := recr.Location(); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if got, want := loc.String(), "/foo/bar/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r.Handle(http.MethodGet, "/foo/bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "_ /foo/bar")
	}))

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "_ /foo/bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodHead, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusMethodNotAllowed; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Method Not Allowed\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	r.Handle("custom", "/barfoo/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "custom /barfoo/*")
	}))

	req = httptest.NewRequest("custom", "/barfoo/", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "custom /barfoo/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	r.Handle("", "/barfoo/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "_ /barfoo/*")
	}))

	for _, method := range []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	} {
		req = httptest.NewRequest(method, "/barfoo/", nil)
		rec = httptest.NewRecorder()
		mh, req = r.Handler(req)
		mh.ServeHTTP(rec, req)
		recr = rec.Result()
		if want := http.StatusOK; recr.StatusCode != want {
			t.Errorf("got %d, want %d", recr.StatusCode, want)
		} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
			t.Fatalf("unexpected error %q", err)
		} else if want := "_ /barfoo/*"; string(b) != want {
			t.Errorf("got %q, want %q", b, want)
		} else if got, want := PathParam(req, "*"), ""; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}

	r.Handle(http.MethodGet, "/bar/foo*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /bar/foo*")
	}))

	req = httptest.NewRequest(http.MethodGet, "/bar/foo", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /bar/foo*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/bar/", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "bar/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRouterHandler_static_param(t *testing.T) {
	r := &Router{}
	r.Handle(http.MethodGet, "/foo", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foo")
	}))
	r.Handle(http.MethodGet, "/foo/:bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foo/:bar")
	}))

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	rec := httptest.NewRecorder()
	mh, req := r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "bar"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/bar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusNotFound; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "Not Found\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}
}

func TestRouterHandler_static_param_wildcardParam(t *testing.T) {
	r := &Router{}
	r.Handle(http.MethodGet, "/", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /")
	}))
	r.Handle(http.MethodGet, "/foo", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foo")
	}))
	r.Handle(http.MethodGet, "/bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /bar")
	}))
	r.Handle(http.MethodGet, "/foobar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foobar")
	}))
	r.Handle(http.MethodGet, "/:foobar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /:foobar")
	}))
	r.Handle(http.MethodGet, "/foo/:bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foo/:bar")
	}))
	r.Handle(http.MethodGet, "/foo:bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foo:bar")
	}))
	r.Handle(http.MethodGet, "/:foo/:bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /:foo/:bar")
	}))
	r.Handle(http.MethodGet, "/:foo/foobar/:bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /:foo/foobar/:bar")
	}))
	r.Handle(http.MethodGet, "/:foo/foobar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /:foo/foobar")
	}))
	r.Handle(http.MethodGet, "/foobar*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foobar*")
	}))
	r.Handle(http.MethodGet, "/foobar/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foobar/*")
	}))
	r.Handle(http.MethodGet, "/foo/:bar/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foo/:bar/*")
	}))
	r.Handle(http.MethodGet, "/foo:bar/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /foo:bar/*")
	}))
	r.Handle(http.MethodGet, "/:foo/:bar/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /:foo/:bar/*")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mh, req := r.Handler(req)
	mh.ServeHTTP(rec, req)
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
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
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
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
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
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
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
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foobar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "foobar"), "barfoo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "bar"), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "bar"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/fooobar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "bar"), "obar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/bar/foo", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foo/:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "foo"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := PathParam(req, "bar"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/bar/foobar/foo", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foo/foobar/:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "foo"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := PathParam(req, "bar"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/bar/foobar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foo/foobar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "foo"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobarfoobar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar/foobar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foobar/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar/foobar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo/:bar/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "bar"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := PathParam(req, "*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foofoobar/foobar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /foo:bar/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "bar"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := PathParam(req, "*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/bar/foo/foobar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foo/:bar/*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "foo"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := PathParam(req, "bar"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := PathParam(req, "*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRouterHandler_fallback(t *testing.T) {
	r := &Router{}
	r.Handle(http.MethodGet, "/*", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /*")
	}))
	r.Handle(http.MethodGet, "/:foo/:bar", http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, "GET /:foo/:bar")
	}))

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	rec := httptest.NewRecorder()
	mh, req := r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr := rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "foobar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /:foo/:bar"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "foo"), "foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	} else if got, want := PathParam(req, "bar"), "bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	req = httptest.NewRequest(http.MethodGet, "/foo/bar/foo", nil)
	rec = httptest.NewRecorder()
	mh, req = r.Handler(req)
	mh.ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusOK; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "GET /*"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	} else if got, want := PathParam(req, "*"), "foo/bar/foo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRouterHandler_misc(t *testing.T) {
	type route struct {
		method string
		path   string
	}

	var routes = []*route{
		{"DELETE", "/1/classes/:className/:objectId"},
		{"DELETE", "/1/installations/:objectId"},
		{"DELETE", "/1/roles/:objectId"},
		{"DELETE", "/1/users/:objectId"},
		{"DELETE", "/applications/:client_id/tokens"},
		{"DELETE", "/applications/:client_id/tokens/:access_token"},
		{"DELETE", "/authorizations/:id"},
		{"DELETE", "/gists/:id"},
		{"DELETE", "/gists/:id/star"},
		{"DELETE", "/moments/:id"},
		{"DELETE", "/notifications/threads/:id/subscription"},
		{"DELETE", "/orgs/:org/members/:user"},
		{"DELETE", "/orgs/:org/public_members/:user"},
		{"DELETE", "/repos/:owner/:repo"},
		{"DELETE", "/repos/:owner/:repo/collaborators/:user"},
		{"DELETE", "/repos/:owner/:repo/comments/:id"},
		{"DELETE", "/repos/:owner/:repo/contents/*"},
		{"DELETE", "/repos/:owner/:repo/downloads/:id"},
		{"DELETE", "/repos/:owner/:repo/git/refs/*"},
		{"DELETE", "/repos/:owner/:repo/hooks/:id"},
		{"DELETE", "/repos/:owner/:repo/issues/:number/labels"},
		{"DELETE", "/repos/:owner/:repo/issues/:number/labels/:name"},
		{"DELETE", "/repos/:owner/:repo/issues/comments/:id"},
		{"DELETE", "/repos/:owner/:repo/keys/:id"},
		{"DELETE", "/repos/:owner/:repo/labels/:name"},
		{"DELETE", "/repos/:owner/:repo/milestones/:number"},
		{"DELETE", "/repos/:owner/:repo/pulls/comments/:number"},
		{"DELETE", "/repos/:owner/:repo/releases/:id"},
		{"DELETE", "/repos/:owner/:repo/subscription"},
		{"DELETE", "/teams/:id"},
		{"DELETE", "/teams/:id/members/:user"},
		{"DELETE", "/teams/:id/repos/:owner/:repo"},
		{"DELETE", "/user/emails"},
		{"DELETE", "/user/following/:user"},
		{"DELETE", "/user/keys/:id"},
		{"DELETE", "/user/starred/:owner/:repo"},
		{"DELETE", "/user/subscriptions/:owner/:repo"},
		{"GET", "/"},
		{"GET", "/1/classes/:className"},
		{"GET", "/1/classes/:className/:objectId"},
		{"GET", "/1/installations"},
		{"GET", "/1/installations/:objectId"},
		{"GET", "/1/login"},
		{"GET", "/1/roles"},
		{"GET", "/1/roles/:objectId"},
		{"GET", "/1/users"},
		{"GET", "/1/users/:objectId"},
		{"GET", "/Makefile"},
		{"GET", "/activities"},
		{"GET", "/activities/:activityId"},
		{"GET", "/activities/:activityId/comments"},
		{"GET", "/activities/:activityId/people/:collection"},
		{"GET", "/applications/:client_id/tokens/:access_token"},
		{"GET", "/articles/"},
		{"GET", "/articles/go_command.html"},
		{"GET", "/articles/index.html"},
		{"GET", "/articles/wiki/"},
		{"GET", "/articles/wiki/Makefile"},
		{"GET", "/articles/wiki/edit.html"},
		{"GET", "/articles/wiki/final-noclosure.go"},
		{"GET", "/articles/wiki/final-noerror.go"},
		{"GET", "/articles/wiki/final-parsetemplate.go"},
		{"GET", "/articles/wiki/final-template.go"},
		{"GET", "/articles/wiki/final.go"},
		{"GET", "/articles/wiki/get.go"},
		{"GET", "/articles/wiki/http-sample.go"},
		{"GET", "/articles/wiki/index.html"},
		{"GET", "/articles/wiki/notemplate.go"},
		{"GET", "/articles/wiki/part1-noerror.go"},
		{"GET", "/articles/wiki/part1.go"},
		{"GET", "/articles/wiki/part2.go"},
		{"GET", "/articles/wiki/part3-errorhandling.go"},
		{"GET", "/articles/wiki/part3.go"},
		{"GET", "/articles/wiki/test.bash"},
		{"GET", "/articles/wiki/test_Test.txt.good"},
		{"GET", "/articles/wiki/test_edit.good"},
		{"GET", "/articles/wiki/test_view.good"},
		{"GET", "/articles/wiki/view.html"},
		{"GET", "/authorizations"},
		{"GET", "/authorizations/:id"},
		{"GET", "/cmd.html"},
		{"GET", "/code.html"},
		{"GET", "/codewalk/"},
		{"GET", "/codewalk/codewalk.css"},
		{"GET", "/codewalk/codewalk.js"},
		{"GET", "/codewalk/codewalk.xml"},
		{"GET", "/codewalk/functions.xml"},
		{"GET", "/codewalk/markov.go"},
		{"GET", "/codewalk/markov.xml"},
		{"GET", "/codewalk/pig.go"},
		{"GET", "/codewalk/popout.png"},
		{"GET", "/codewalk/run"},
		{"GET", "/codewalk/sharemem.xml"},
		{"GET", "/codewalk/urlpoll.go"},
		{"GET", "/comments/:commentId"},
		{"GET", "/contrib.html"},
		{"GET", "/contribute.html"},
		{"GET", "/debugging_with_gdb.html"},
		{"GET", "/devel/"},
		{"GET", "/devel/release.html"},
		{"GET", "/devel/weekly.html"},
		{"GET", "/docs.html"},
		{"GET", "/effective_go.html"},
		{"GET", "/emojis"},
		{"GET", "/events"},
		{"GET", "/feeds"},
		{"GET", "/files.log"},
		{"GET", "/gccgo_contribute.html"},
		{"GET", "/gccgo_install.html"},
		{"GET", "/gists"},
		{"GET", "/gists/:id"},
		{"GET", "/gists/:id/star"},
		{"GET", "/gists/public"},
		{"GET", "/gists/starred"},
		{"GET", "/gitignore/templates"},
		{"GET", "/gitignore/templates/:name"},
		{"GET", "/go-logo-black.png"},
		{"GET", "/go-logo-blue.png"},
		{"GET", "/go-logo-white.png"},
		{"GET", "/go1.1.html"},
		{"GET", "/go1.2.html"},
		{"GET", "/go1.html"},
		{"GET", "/go1compat.html"},
		{"GET", "/go_faq.html"},
		{"GET", "/go_mem.html"},
		{"GET", "/go_spec.html"},
		{"GET", "/gopher/"},
		{"GET", "/gopher/appenginegopher.jpg"},
		{"GET", "/gopher/appenginegophercolor.jpg"},
		{"GET", "/gopher/appenginelogo.gif"},
		{"GET", "/gopher/bumper.png"},
		{"GET", "/gopher/bumper192x108.png"},
		{"GET", "/gopher/bumper320x180.png"},
		{"GET", "/gopher/bumper480x270.png"},
		{"GET", "/gopher/bumper640x360.png"},
		{"GET", "/gopher/doc.png"},
		{"GET", "/gopher/frontpage.png"},
		{"GET", "/gopher/gopherbw.png"},
		{"GET", "/gopher/gophercolor.png"},
		{"GET", "/gopher/gophercolor16x16.png"},
		{"GET", "/gopher/help.png"},
		{"GET", "/gopher/pencil/"},
		{"GET", "/gopher/pencil/gopherhat.jpg"},
		{"GET", "/gopher/pencil/gopherhelmet.jpg"},
		{"GET", "/gopher/pencil/gophermega.jpg"},
		{"GET", "/gopher/pencil/gopherrunning.jpg"},
		{"GET", "/gopher/pencil/gopherswim.jpg"},
		{"GET", "/gopher/pencil/gopherswrench.jpg"},
		{"GET", "/gopher/pkg.png"},
		{"GET", "/gopher/project.png"},
		{"GET", "/gopher/ref.png"},
		{"GET", "/gopher/run.png"},
		{"GET", "/gopher/talks.png"},
		{"GET", "/help.html"},
		{"GET", "/ie.css"},
		{"GET", "/install-source.html"},
		{"GET", "/install.html"},
		{"GET", "/issues"},
		{"GET", "/legacy/issues/search/:owner/:repo/:state/:keyword"},
		{"GET", "/legacy/repos/search/:keyword"},
		{"GET", "/legacy/user/email/:email"},
		{"GET", "/legacy/user/search/:keyword"},
		{"GET", "/logo-153x55.png"},
		{"GET", "/meta"},
		{"GET", "/networks/:owner/:repo/events"},
		{"GET", "/notifications"},
		{"GET", "/notifications/threads/:id"},
		{"GET", "/notifications/threads/:id/subscription"},
		{"GET", "/orgs/:org"},
		{"GET", "/orgs/:org/events"},
		{"GET", "/orgs/:org/issues"},
		{"GET", "/orgs/:org/members"},
		{"GET", "/orgs/:org/members/:user"},
		{"GET", "/orgs/:org/public_members"},
		{"GET", "/orgs/:org/public_members/:user"},
		{"GET", "/orgs/:org/repos"},
		{"GET", "/orgs/:org/teams"},
		{"GET", "/people"},
		{"GET", "/people/:userId"},
		{"GET", "/people/:userId/activities/:collection"},
		{"GET", "/people/:userId/moments/:collection"},
		{"GET", "/people/:userId/openIdConnect"},
		{"GET", "/people/:userId/people/:collection"},
		{"GET", "/play/"},
		{"GET", "/play/fib.go"},
		{"GET", "/play/hello.go"},
		{"GET", "/play/life.go"},
		{"GET", "/play/peano.go"},
		{"GET", "/play/pi.go"},
		{"GET", "/play/sieve.go"},
		{"GET", "/play/solitaire.go"},
		{"GET", "/play/tree.go"},
		{"GET", "/progs/"},
		{"GET", "/progs/cgo1.go"},
		{"GET", "/progs/cgo2.go"},
		{"GET", "/progs/cgo3.go"},
		{"GET", "/progs/cgo4.go"},
		{"GET", "/progs/defer.go"},
		{"GET", "/progs/defer.out"},
		{"GET", "/progs/defer2.go"},
		{"GET", "/progs/defer2.out"},
		{"GET", "/progs/eff_bytesize.go"},
		{"GET", "/progs/eff_bytesize.out"},
		{"GET", "/progs/eff_qr.go"},
		{"GET", "/progs/eff_sequence.go"},
		{"GET", "/progs/eff_sequence.out"},
		{"GET", "/progs/eff_unused1.go"},
		{"GET", "/progs/eff_unused2.go"},
		{"GET", "/progs/error.go"},
		{"GET", "/progs/error2.go"},
		{"GET", "/progs/error3.go"},
		{"GET", "/progs/error4.go"},
		{"GET", "/progs/go1.go"},
		{"GET", "/progs/gobs1.go"},
		{"GET", "/progs/gobs2.go"},
		{"GET", "/progs/image_draw.go"},
		{"GET", "/progs/image_package1.go"},
		{"GET", "/progs/image_package1.out"},
		{"GET", "/progs/image_package2.go"},
		{"GET", "/progs/image_package2.out"},
		{"GET", "/progs/image_package3.go"},
		{"GET", "/progs/image_package3.out"},
		{"GET", "/progs/image_package4.go"},
		{"GET", "/progs/image_package4.out"},
		{"GET", "/progs/image_package5.go"},
		{"GET", "/progs/image_package5.out"},
		{"GET", "/progs/image_package6.go"},
		{"GET", "/progs/image_package6.out"},
		{"GET", "/progs/interface.go"},
		{"GET", "/progs/interface2.go"},
		{"GET", "/progs/interface2.out"},
		{"GET", "/progs/json1.go"},
		{"GET", "/progs/json2.go"},
		{"GET", "/progs/json2.out"},
		{"GET", "/progs/json3.go"},
		{"GET", "/progs/json4.go"},
		{"GET", "/progs/json5.go"},
		{"GET", "/progs/run"},
		{"GET", "/progs/slices.go"},
		{"GET", "/progs/timeout1.go"},
		{"GET", "/progs/timeout2.go"},
		{"GET", "/progs/update.bash"},
		{"GET", "/rate_limit"},
		{"GET", "/repos/:owner/:repo"},
		{"GET", "/repos/:owner/:repo/:archive_format/:ref"},
		{"GET", "/repos/:owner/:repo/assignees"},
		{"GET", "/repos/:owner/:repo/assignees/:assignee"},
		{"GET", "/repos/:owner/:repo/branches"},
		{"GET", "/repos/:owner/:repo/branches/:branch"},
		{"GET", "/repos/:owner/:repo/collaborators"},
		{"GET", "/repos/:owner/:repo/collaborators/:user"},
		{"GET", "/repos/:owner/:repo/comments"},
		{"GET", "/repos/:owner/:repo/comments/:id"},
		{"GET", "/repos/:owner/:repo/commits"},
		{"GET", "/repos/:owner/:repo/commits/:sha"},
		{"GET", "/repos/:owner/:repo/commits/:sha/comments"},
		{"GET", "/repos/:owner/:repo/contents/*"},
		{"GET", "/repos/:owner/:repo/contributors"},
		{"GET", "/repos/:owner/:repo/downloads"},
		{"GET", "/repos/:owner/:repo/downloads/:id"},
		{"GET", "/repos/:owner/:repo/events"},
		{"GET", "/repos/:owner/:repo/forks"},
		{"GET", "/repos/:owner/:repo/git/blobs/:sha"},
		{"GET", "/repos/:owner/:repo/git/commits/:sha"},
		{"GET", "/repos/:owner/:repo/git/refs"},
		{"GET", "/repos/:owner/:repo/git/refs/*"},
		{"GET", "/repos/:owner/:repo/git/tags/:sha"},
		{"GET", "/repos/:owner/:repo/git/trees/:sha"},
		{"GET", "/repos/:owner/:repo/hooks"},
		{"GET", "/repos/:owner/:repo/hooks/:id"},
		{"GET", "/repos/:owner/:repo/issues"},
		{"GET", "/repos/:owner/:repo/issues/:number"},
		{"GET", "/repos/:owner/:repo/issues/:number/comments"},
		{"GET", "/repos/:owner/:repo/issues/:number/events"},
		{"GET", "/repos/:owner/:repo/issues/:number/labels"},
		{"GET", "/repos/:owner/:repo/issues/comments"},
		{"GET", "/repos/:owner/:repo/issues/comments/:id"},
		{"GET", "/repos/:owner/:repo/issues/events"},
		{"GET", "/repos/:owner/:repo/issues/events/:id"},
		{"GET", "/repos/:owner/:repo/keys"},
		{"GET", "/repos/:owner/:repo/keys/:id"},
		{"GET", "/repos/:owner/:repo/labels"},
		{"GET", "/repos/:owner/:repo/labels/:name"},
		{"GET", "/repos/:owner/:repo/languages"},
		{"GET", "/repos/:owner/:repo/milestones"},
		{"GET", "/repos/:owner/:repo/milestones/:number"},
		{"GET", "/repos/:owner/:repo/milestones/:number/labels"},
		{"GET", "/repos/:owner/:repo/notifications"},
		{"GET", "/repos/:owner/:repo/pulls"},
		{"GET", "/repos/:owner/:repo/pulls/:number"},
		{"GET", "/repos/:owner/:repo/pulls/:number/comments"},
		{"GET", "/repos/:owner/:repo/pulls/:number/commits"},
		{"GET", "/repos/:owner/:repo/pulls/:number/files"},
		{"GET", "/repos/:owner/:repo/pulls/:number/merge"},
		{"GET", "/repos/:owner/:repo/pulls/comments"},
		{"GET", "/repos/:owner/:repo/pulls/comments/:number"},
		{"GET", "/repos/:owner/:repo/readme"},
		{"GET", "/repos/:owner/:repo/releases"},
		{"GET", "/repos/:owner/:repo/releases/:id"},
		{"GET", "/repos/:owner/:repo/releases/:id/assets"},
		{"GET", "/repos/:owner/:repo/stargazers"},
		{"GET", "/repos/:owner/:repo/stats/code_frequency"},
		{"GET", "/repos/:owner/:repo/stats/commit_activity"},
		{"GET", "/repos/:owner/:repo/stats/contributors"},
		{"GET", "/repos/:owner/:repo/stats/participation"},
		{"GET", "/repos/:owner/:repo/stats/punch_card"},
		{"GET", "/repos/:owner/:repo/statuses/:ref"},
		{"GET", "/repos/:owner/:repo/subscribers"},
		{"GET", "/repos/:owner/:repo/subscription"},
		{"GET", "/repos/:owner/:repo/tags"},
		{"GET", "/repos/:owner/:repo/teams"},
		{"GET", "/repositories"},
		{"GET", "/root.html"},
		{"GET", "/search/code"},
		{"GET", "/search/issues"},
		{"GET", "/search/repositories"},
		{"GET", "/search/users"},
		{"GET", "/share.png"},
		{"GET", "/sieve.gif"},
		{"GET", "/teams/:id"},
		{"GET", "/teams/:id/members"},
		{"GET", "/teams/:id/members/:user"},
		{"GET", "/teams/:id/repos"},
		{"GET", "/teams/:id/repos/:owner/:repo"},
		{"GET", "/tos.html"},
		{"GET", "/user"},
		{"GET", "/user/emails"},
		{"GET", "/user/followers"},
		{"GET", "/user/following"},
		{"GET", "/user/following/:user"},
		{"GET", "/user/issues"},
		{"GET", "/user/keys"},
		{"GET", "/user/keys/:id"},
		{"GET", "/user/orgs"},
		{"GET", "/user/repos"},
		{"GET", "/user/starred"},
		{"GET", "/user/starred/:owner/:repo"},
		{"GET", "/user/subscriptions"},
		{"GET", "/user/subscriptions/:owner/:repo"},
		{"GET", "/user/teams"},
		{"GET", "/users"},
		{"GET", "/users/:user"},
		{"GET", "/users/:user/events"},
		{"GET", "/users/:user/events/orgs/:org"},
		{"GET", "/users/:user/events/public"},
		{"GET", "/users/:user/followers"},
		{"GET", "/users/:user/following"},
		{"GET", "/users/:user/following/:target_user"},
		{"GET", "/users/:user/gists"},
		{"GET", "/users/:user/keys"},
		{"GET", "/users/:user/orgs"},
		{"GET", "/users/:user/received_events"},
		{"GET", "/users/:user/received_events/public"},
		{"GET", "/users/:user/repos"},
		{"GET", "/users/:user/starred"},
		{"GET", "/users/:user/subscriptions"},
		{"PATCH", "/authorizations/:id"},
		{"PATCH", "/gists/:id"},
		{"PATCH", "/notifications/threads/:id"},
		{"PATCH", "/orgs/:org"},
		{"PATCH", "/repos/:owner/:repo"},
		{"PATCH", "/repos/:owner/:repo/comments/:id"},
		{"PATCH", "/repos/:owner/:repo/git/refs/*"},
		{"PATCH", "/repos/:owner/:repo/hooks/:id"},
		{"PATCH", "/repos/:owner/:repo/issues/:number"},
		{"PATCH", "/repos/:owner/:repo/issues/comments/:id"},
		{"PATCH", "/repos/:owner/:repo/keys/:id"},
		{"PATCH", "/repos/:owner/:repo/labels/:name"},
		{"PATCH", "/repos/:owner/:repo/milestones/:number"},
		{"PATCH", "/repos/:owner/:repo/pulls/:number"},
		{"PATCH", "/repos/:owner/:repo/pulls/comments/:number"},
		{"PATCH", "/repos/:owner/:repo/releases/:id"},
		{"PATCH", "/teams/:id"},
		{"PATCH", "/user"},
		{"PATCH", "/user/keys/:id"},
		{"POST", "/1/classes/:className"},
		{"POST", "/1/events/:eventName"},
		{"POST", "/1/files/:fileName"},
		{"POST", "/1/functions"},
		{"POST", "/1/installations"},
		{"POST", "/1/push"},
		{"POST", "/1/requestPasswordReset"},
		{"POST", "/1/roles"},
		{"POST", "/1/users"},
		{"POST", "/authorizations"},
		{"POST", "/gists"},
		{"POST", "/gists/:id/forks"},
		{"POST", "/markdown"},
		{"POST", "/markdown/raw"},
		{"POST", "/orgs/:org/repos"},
		{"POST", "/orgs/:org/teams"},
		{"POST", "/people/:userId/moments/:collection"},
		{"POST", "/repos/:owner/:repo/commits/:sha/comments"},
		{"POST", "/repos/:owner/:repo/forks"},
		{"POST", "/repos/:owner/:repo/git/blobs"},
		{"POST", "/repos/:owner/:repo/git/commits"},
		{"POST", "/repos/:owner/:repo/git/refs"},
		{"POST", "/repos/:owner/:repo/git/tags"},
		{"POST", "/repos/:owner/:repo/git/trees"},
		{"POST", "/repos/:owner/:repo/hooks"},
		{"POST", "/repos/:owner/:repo/hooks/:id/tests"},
		{"POST", "/repos/:owner/:repo/issues"},
		{"POST", "/repos/:owner/:repo/issues/:number/comments"},
		{"POST", "/repos/:owner/:repo/issues/:number/labels"},
		{"POST", "/repos/:owner/:repo/keys"},
		{"POST", "/repos/:owner/:repo/labels"},
		{"POST", "/repos/:owner/:repo/merges"},
		{"POST", "/repos/:owner/:repo/milestones"},
		{"POST", "/repos/:owner/:repo/pulls"},
		{"POST", "/repos/:owner/:repo/releases"},
		{"POST", "/repos/:owner/:repo/statuses/:ref"},
		{"POST", "/user/emails"},
		{"POST", "/user/keys"},
		{"POST", "/user/repos"},
		{"PUT", "/1/classes/:className/:objectId"},
		{"PUT", "/1/installations/:objectId"},
		{"PUT", "/1/roles/:objectId"},
		{"PUT", "/1/users/:objectId"},
		{"PUT", "/authorizations/clients/:client_id"},
		{"PUT", "/gists/:id/star"},
		{"PUT", "/notifications"},
		{"PUT", "/notifications/threads/:id/subscription"},
		{"PUT", "/orgs/:org/public_members/:user"},
		{"PUT", "/repos/:owner/:repo/collaborators/:user"},
		{"PUT", "/repos/:owner/:repo/contents/*"},
		{"PUT", "/repos/:owner/:repo/issues/:number/labels"},
		{"PUT", "/repos/:owner/:repo/notifications"},
		{"PUT", "/repos/:owner/:repo/pulls/:number/comments"},
		{"PUT", "/repos/:owner/:repo/pulls/:number/merge"},
		{"PUT", "/repos/:owner/:repo/subscription"},
		{"PUT", "/teams/:id/members/:user"},
		{"PUT", "/teams/:id/repos/:owner/:repo"},
		{"PUT", "/user/following/:user"},
		{"PUT", "/user/starred/:owner/:repo"},
		{"PUT", "/user/subscriptions/:owner/:repo"},
	}

	r := &Router{}
	for _, route := range routes {
		routeName := fmt.Sprintf("%s %s", route.method, route.path)
		r.Handle(route.method, route.path, http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			fmt.Fprint(rw, routeName)
		}))
	}

	for _, route := range routes {
		routeName := fmt.Sprintf("%s %s", route.method, route.path)
		data := &data{}
		for i, l := 0, len(route.path); i < l; i++ {
			switch route.path[i] {
			case ':':
				j := i + 1
				for ; i < l && route.path[i] != '/'; i++ {
				}

				data.pathParamNames = append(
					data.pathParamNames,
					route.path[j:i],
				)
				data.pathParamValues = append(
					data.pathParamValues,
					route.path[j-1:i],
				)
			case '*':
				data.pathParamNames = append(
					data.pathParamNames,
					"*",
				)
				data.pathParamValues = append(
					data.pathParamValues,
					"*",
				)
			}
		}

		req := httptest.NewRequest(route.method, route.path, nil)
		rec := httptest.NewRecorder()
		mh, req := r.Handler(req)
		mh.ServeHTTP(rec, req)
		recr := rec.Result()
		if want := http.StatusOK; recr.StatusCode != want {
			t.Errorf("got %d, want %d", recr.StatusCode, want)
		} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
			t.Fatalf("unexpected error %q", err)
		} else if string(b) != routeName {
			t.Errorf("got %q, want %q", b, want)
		} else if l := len(data.pathParamNames); l > 0 {
			ppns := PathParamNames(req)
			if ppns == nil {
				t.Fatal("unexpected nil")
			} else if got := len(ppns); got != l {
				t.Errorf("got %d, want %d", got, l)
			}

			for i, ppn := range ppns {
				want := data.pathParamNames[i]
				if ppn != want {
					t.Errorf("got %q, want %q", ppn, want)
				}
			}
		} else if l := len(data.pathParamValues); l > 0 {
			ppvs := PathParamValues(req)
			if ppvs == nil {
				t.Fatal("unexpected nil")
			}

			for i, ppv := range ppvs {
				want := data.pathParamValues[i]
				if ppv != want {
					t.Errorf("got %q, want %q", ppv, want)
				}
			}
		}
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
			methodHandlerSet: &methodHandlerSet{},
		},
		registeredRoutes: map[string]bool{},
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
		Middlewares: []Middleware{MiddlewareFunc(func(
			next http.Handler,
		) http.Handler {
			return http.HandlerFunc(func(
				rw http.ResponseWriter,
				req *http.Request,
			) {
				next.ServeHTTP(rw, req)
				fmt.Fprint(rw, "middleware")
			})
		})},
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
	} else if want := "custom\nmiddleware"; string(b) != want {
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
		Middlewares: []Middleware{MiddlewareFunc(func(
			next http.Handler,
		) http.Handler {
			return http.HandlerFunc(func(
				rw http.ResponseWriter,
				req *http.Request,
			) {
				next.ServeHTTP(rw, req)
				fmt.Fprint(rw, "middleware")
			})
		})},
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
	} else if want := "custom\nmiddleware"; string(b) != want {
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
		Middlewares: []Middleware{MiddlewareFunc(func(
			next http.Handler,
		) http.Handler {
			return http.HandlerFunc(func(
				rw http.ResponseWriter,
				req *http.Request,
			) {
				next.ServeHTTP(rw, req)
				fmt.Fprint(rw, "middleware")
			})
		})},
		TSRHandler: http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			http.Error(rw, "custom", http.StatusBadRequest)
		}),
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	r.tsrHandler().ServeHTTP(rec, req)
	recr = rec.Result()
	if want := http.StatusBadRequest; recr.StatusCode != want {
		t.Errorf("got %d, want %d", recr.StatusCode, want)
	} else if b, err := ioutil.ReadAll(recr.Body); err != nil {
		t.Fatalf("unexpected error %q", err)
	} else if want := "custom\nmiddleware"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}
}

func TestRouteNodeAddChild(t *testing.T) {
	rn := &routeNode{}
	rn.addChild(&routeNode{
		label: 'a',
		typ:   staticRouteNode,
	})
	rn.addChild(&routeNode{
		label: ':',
		typ:   paramRouteNode,
	})
	rn.addChild(&routeNode{
		label: '*',
		typ:   wildcardParamRouteNode,
	})
	if got, want := len(rn.staticChildren), 1; got != want {
		t.Errorf("got %d, want %d", got, want)
	} else if rn.paramChild == nil {
		t.Fatal("unexpected nil")
	} else if rn.wildcardParamChild == nil {
		t.Fatal("unexpected nil")
	} else if !rn.hasAtLeastOneChild {
		t.Error("want true")
	}
}

func TestRouteNodeSetHandler(t *testing.T) {
	h := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
	})

	rn := &routeNode{
		methodHandlerSet: &methodHandlerSet{},
	}

	rn.setHandler(http.MethodGet, h)
	if rn.methodHandlerSet.get == nil {
		t.Fatal("unexpected nil")
	}

	rn = &routeNode{
		methodHandlerSet: &methodHandlerSet{},
	}

	rn.setHandler(http.MethodHead, h)
	if rn.methodHandlerSet.head == nil {
		t.Fatal("unexpected nil")
	}

	rn = &routeNode{
		methodHandlerSet: &methodHandlerSet{},
	}

	rn.setHandler(http.MethodPost, h)
	if rn.methodHandlerSet.post == nil {
		t.Fatal("unexpected nil")
	}

	rn = &routeNode{
		methodHandlerSet: &methodHandlerSet{},
	}

	rn.setHandler(http.MethodPut, h)
	if rn.methodHandlerSet.put == nil {
		t.Fatal("unexpected nil")
	}

	rn = &routeNode{
		methodHandlerSet: &methodHandlerSet{},
	}

	rn.setHandler(http.MethodPatch, h)
	if rn.methodHandlerSet.patch == nil {
		t.Fatal("unexpected nil")
	}

	rn = &routeNode{
		methodHandlerSet: &methodHandlerSet{},
	}

	rn.setHandler(http.MethodDelete, h)
	if rn.methodHandlerSet.delete == nil {
		t.Fatal("unexpected nil")
	}

	rn = &routeNode{
		methodHandlerSet: &methodHandlerSet{},
	}

	rn.setHandler(http.MethodConnect, h)
	if rn.methodHandlerSet.connect == nil {
		t.Fatal("unexpected nil")
	}

	rn = &routeNode{
		methodHandlerSet: &methodHandlerSet{},
	}

	rn.setHandler(http.MethodOptions, h)
	if rn.methodHandlerSet.options == nil {
		t.Fatal("unexpected nil")
	}

	rn = &routeNode{
		methodHandlerSet: &methodHandlerSet{},
	}

	rn.setHandler(http.MethodTrace, h)
	if rn.methodHandlerSet.trace == nil {
		t.Fatal("unexpected nil")
	}

	rn = &routeNode{
		methodHandlerSet: &methodHandlerSet{},
	}

	rn.setHandler("foobar", h)
	if got, want := len(rn.otherMethodHandlers), 1; got != want {
		t.Errorf("got %d, want %d", got, want)
	}

	rn.otherMethodHandlers[0].handler = nil
	rn.setHandler("foobar", h)
	if rn.otherMethodHandlers[0].handler == nil {
		t.Fatal("unexpected nil")
	}

	rn.setHandler("foobar", nil)
	if got, want := len(rn.otherMethodHandlers), 0; got != want {
		t.Errorf("got %d, want %d", got, want)
	}

	rn = &routeNode{
		methodHandlerSet: &methodHandlerSet{},
	}

	rn.setHandler("", h)
	if rn.catchAllHandler == nil {
		t.Fatal("unexpected nil")
	}
}
