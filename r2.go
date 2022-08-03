/*
Package r2 implements a minimalist HTTP request routing helper for Go.
*/
package r2

import (
	"context"
	"net/http"
	"time"
)

// Context returns a non-nil [context.Context] that is never canceled and has no
// deadline. It is typically used in the [http.Server.BaseContext] to avoid the
// [Router.Handler] from calling the [http.Request.WithContext], which can
// significantly improve routing performance.
func Context() context.Context {
	return &dataContext{d: &data{}}
}

// PathParam returns a path parameter value of the req for the name. It returns
// empty string if not found.
func PathParam(req *http.Request, name string) string {
	d, ok := req.Context().Value(dataContextKey).(*data)
	if !ok {
		return ""
	}

	for i, ppn := range d.pathParamNames {
		if ppn == name {
			return d.pathParamValues[i]
		}
	}

	return ""
}

// PathParamNames returns path parameter names of the req. It returns nil if not
// found.
func PathParamNames(req *http.Request) []string {
	d, ok := req.Context().Value(dataContextKey).(*data)
	if !ok {
		return nil
	}

	return d.pathParamNames
}

// PathParamValues returns path parameter values of the req. It returns nil if
// not found.
func PathParamValues(req *http.Request) []string {
	d, ok := req.Context().Value(dataContextKey).(*data)
	if !ok {
		return nil
	}

	return d.pathParamValues[:len(d.pathParamNames)]
}

// dataContext is a [context.Context] that wraps a [data].
type dataContext struct {
	d *data
}

// Deadline implements the [context.Context].
func (*dataContext) Deadline() (deadline time.Time, ok bool) { return }

// Done implements the [context.Context].
func (*dataContext) Done() <-chan struct{} { return nil }

// Err implements the [context.Context].
func (*dataContext) Err() error { return nil }

// Value implements the [context.Context].
func (dc *dataContext) Value(key interface{}) interface{} {
	if key == dataContextKey {
		return dc.d
	}

	return nil
}

// contextKey is a key for a context value.
type contextKey uint8

// The context keys.
const (
	dataContextKey contextKey = iota
)

// data is a request-scoped data set.
type data struct {
	pathParamNames  []string
	pathParamValues []string
}
