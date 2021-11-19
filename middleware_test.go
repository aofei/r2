package r2

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareFuncChainHTTPHandler(t *testing.T) {
	mf := MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			req.Host = "www.example.com"
			next.ServeHTTP(rw, req)
		})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	mf.ChainHTTPHandler(http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		fmt.Fprint(rw, req.Host)
	})).ServeHTTP(rec, req)

	recb := rec.Body.String()
	if want := "www.example.com"; recb != want {
		t.Errorf("got %q, want %q", recb, want)
	}
}
