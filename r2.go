/*
Package r2 implements a minimalist HTTP request routing helper for Go.
*/
package r2

import "net/http"

// contextKey is a key for use with the `context.WithValue`.
type contextKey uint8

// The context keys.
const (
	pathParamsContextKey contextKey = iota
)

// PathParam returns a path parameter value of the `req` for the `name`.
func PathParam(req *http.Request, name string) string {
	pps, ok := req.Context().Value(pathParamsContextKey).(*pathParams)
	if !ok {
		return ""
	}

	for i, ppn := range pps.names {
		if ppn == name {
			return pps.values[i]
		}
	}

	return ""
}

// PathParamNames returns path parameter names of the `req`.
func PathParamNames(req *http.Request) []string {
	pps, ok := req.Context().Value(pathParamsContextKey).(*pathParams)
	if !ok {
		return nil
	}

	return pps.names
}

// PathParamValues returns path parameter values of the `req`.
func PathParamValues(req *http.Request) []string {
	pps, ok := req.Context().Value(pathParamsContextKey).(*pathParams)
	if !ok {
		return nil
	}

	return pps.values[:len(pps.names)]
}
