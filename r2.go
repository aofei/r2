/*
Package r2 implements a minimalist HTTP request routing helper for Go.
*/
package r2

import (
	"net/http"
	"net/url"
	"strings"
)

// contextKey is the context key.
type contextKey uint8

// The context keys.
const (
	matchDataContextKey contextKey = iota
)

// PathParams returns parsed path parameters of the `req`.
//
// Note that the returned `url.Values` is always non-nil.
func PathParams(req *http.Request) url.Values {
	md, ok := req.Context().Value(matchDataContextKey).(*matchData)
	if !ok {
		return url.Values{}
	}

	if md.pathParams != nil {
		return md.pathParams
	}

	ppsl := len(md.pathParamSlots)
	pps := make(url.Values, ppsl)
	maxPPS := md.pathParamSlots[ppsl-1].number
	for i, l, slot, ppsi := 0, len(md.path), 0, 0; i < l; i++ {
		if md.path[i] == '/' {
			i++
			for ; i < l && md.path[i] == '/'; i++ {
			}

			slot++
			if slot > maxPPS {
				break
			}
		}

		if slot < md.pathParamSlots[ppsi].number {
			j := strings.IndexByte(md.path[i:], '/')
			if j < 0 { // This should never happen
				break
			}

			i += j - 1

			continue
		}

		i += md.pathParamSlots[ppsi].offset

		if md.pathParamSlots[ppsi].name == "*" {
			addPathParam(pps, "*", md.path[i:])
			break
		}

		j := strings.IndexByte(md.path[i:], '/')
		if j < 0 {
			addPathParam(
				pps,
				md.pathParamSlots[ppsi].name,
				md.path[i:],
			)
			break
		}

		addPathParam(pps, md.pathParamSlots[ppsi].name, md.path[i:i+j])

		ppsi++
		if ppsi >= ppsl {
			break
		}

		i += j - 1
	}

	md.pathParams = pps

	return pps
}

// addPathParam adds the `name` and `rawValue` as a path parameter to the `vs`.
func addPathParam(vs url.Values, name, rawValue string) {
	value, err := url.PathUnescape(rawValue)
	if err != nil {
		value = rawValue
	}

	if len(vs[name]) == 0 {
		vs[name] = []string{value}
	} else {
		vs.Add(name, value)
	}
}
