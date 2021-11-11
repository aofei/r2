/*
Package r2 implements a minimalist HTTP request routing helper for Go.
*/
package r2

import (
	"net/http"
	"net/url"
	"strings"
)

// key is the key of the request scoped data.
type key uint8

// The keys.
const (
	dataKey key = iota
	routeNodeKey
	pathKey
	pathParamsKey
	valuesKey
	deferredFuncsKey
)

// data returns request scoped data of the `req`.
func data(req *http.Request) map[interface{}]interface{} {
	if vsi := req.Context().Value(dataKey); vsi != nil {
		return vsi.(map[interface{}]interface{})
	}

	return nil
}

// PathParams returns parsed path parameters of the `req`.
//
// Note that the returned `url.Values` is always non-nil, unless the `req` is
// not from the `http.Handler` returned by the `Router.Handler`.
func PathParams(req *http.Request) url.Values {
	d := data(req)
	if d == nil {
		return nil
	}

	if ppsi, ok := d[pathParamsKey]; ok {
		return ppsi.(url.Values)
	}

	if rn := d[routeNodeKey].(*routeNode); len(rn.pathParamSlots) > 0 {
		pps := make(url.Values, len(rn.pathParamSlots))

		path := d[pathKey].(string)
		for i, l, slot := 0, len(path), 0; i < l; i++ {
			if path[i] == '/' {
				i++
				for ; i < l && path[i] == '/'; i++ {
				}

				slot++
				if slot > rn.maxPathParamSlot {
					break
				}
			}

			s, ok := rn.pathParamSlots[slot]
			if !ok {
				j := strings.IndexByte(path[i:], '/')
				if j > 0 {
					i += j - 1
					continue
				}

				break // This should never happen
			}

			n := s.name
			i += s.offset

			if n == "*" {
				pps.Add(n, unescapePathParamValue(path[i:]))
				break
			}

			if j := strings.IndexByte(path[i:], '/'); j > 0 {
				pps.Add(n, unescapePathParamValue(path[i:i+j]))
				if slot == rn.maxPathParamSlot {
					break
				}

				i += j - 1
			} else {
				pps.Add(n, unescapePathParamValue(path[i:]))
				break
			}
		}

		d[pathParamsKey] = pps

		return pps
	}

	pps := url.Values{}
	d[pathParamsKey] = pps

	return pps
}

// unescapePathParamValue unescapes the `s` as a path parameter value.
func unescapePathParamValue(s string) string {
	if us, err := url.PathUnescape(s); err == nil {
		return us
	}

	return s
}

// Values returns request scoped arbitrary values of the `req`.
//
// Note that the returned `map[interface{}]interface{}` is always non-nil,
// unless the `req` is not from the `http.Handler` returned by the
// `Router.Handler`.
func Values(req *http.Request) map[interface{}]interface{} {
	d := data(req)
	if d == nil {
		return nil
	}

	if vsi, ok := d[valuesKey]; ok {
		return vsi.(map[interface{}]interface{})
	}

	vs := map[interface{}]interface{}{}
	d[valuesKey] = vs

	return vs
}

// Defer pushes the `f` onto the stack of functions that will be called after
// the matched `http.Handler` for the `req` returns.
func Defer(req *http.Request, f func()) {
	d := data(req)
	if d == nil {
		return
	}

	if f == nil {
		return
	}

	if dfsi, ok := d[deferredFuncsKey]; ok {
		d[deferredFuncsKey] = append(dfsi.([]func()), f)
	} else {
		d[deferredFuncsKey] = []func(){f}
	}
}
