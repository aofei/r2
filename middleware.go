package r2

import "net/http"

// Middleware is used to chain the `http.Handler`s.
type Middleware interface {
	// ChainHTTPHandler chains the `next` to the returned `http.Handler`.
	//
	// Typically, the returned `http.Handler` is a closure which does
	// something with the `http.ResponseWriter` and `http.Request` passed to
	// it, and then calls the `next.ServeHTTP`.
	ChainHTTPHandler(next http.Handler) http.Handler
}

// MiddlewareFunc is an adapter to allow the use of an ordinary function as a
// `Middleware`.
type MiddlewareFunc func(next http.Handler) http.Handler

// ChainHTTPHandler implements the `Middleware`.
func (mf MiddlewareFunc) ChainHTTPHandler(next http.Handler) http.Handler {
	return mf(next)
}
