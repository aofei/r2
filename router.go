package r2

import (
	"context"
	"net/http"
	ppath "path"
	"strings"
)

// Router is the registry of all registered routes for request matching.
type Router struct {
	// Parent is the parent `Router`.
	Parent *Router

	// PathPrefix is the path prefix of all routes to be registered.
	PathPrefix string

	// Middlewares is the `Middleware` chain that performs after routing.
	Middlewares []Middleware

	// NotFoundHandler writes 404 not found responses.
	//
	// If the `NotFoundHandler` is nil, a default one is used.
	//
	// Note that the `NotFoundHandler` will be ignored when the `Parent` is
	// not nil.
	NotFoundHandler http.Handler

	// MethodNotAllowedHandler writes 405 method not allowed responses.
	//
	// If the `MethodNotAllowedHandler` is nil, a default one is used.
	//
	// Note that the `MethodNotAllowedHandler` will be ignored when the
	// `Parent` is not nil.
	MethodNotAllowedHandler http.Handler

	// TSRHandler writes TSR (trailing slash redirect) responses.
	//
	// If the `TSRHandler` is nil, a default one is used.
	//
	// Note that the `TSRHandler` will be ignored when the `Parent` is not
	// nil.
	TSRHandler http.Handler

	routeTree         *routeNode
	registeredRoutes  map[string]bool
	overridableRoutes map[string]bool
}

// Sub returns a new instance of the `Router` inherited from the `r` with the
// `pathPrefix` and optional `ms`.
func (r *Router) Sub(pathPrefix string, ms ...Middleware) *Router {
	return &Router{
		Parent:      r,
		PathPrefix:  pathPrefix,
		Middlewares: ms,
	}
}

// Handle registers a new route for the `method` ("" means any) and `path` with
// the matching `h` and optional `ms`.
//
// Note that a ':' followed by a name in the `path` declares a path parameter
// that matches all characters except '/'. And an '*' in the `path` declares a
// wildcard path parameter that greedily matches all characters, with "*" as its
// name. The `PathParams` can be used to get those declared path parameters
// after a request is matched.
func (r *Router) Handle(method, path string, h http.Handler, ms ...Middleware) {
	if r.Parent != nil {
		r.Parent.Handle(
			method,
			r.PathPrefix+path,
			h,
			append(r.Middlewares, ms...)...,
		)
		return
	}

	if r.routeTree == nil {
		r.routeTree = &routeNode{
			handlers: map[string]http.Handler{},
		}

		r.registeredRoutes = map[string]bool{}
		r.overridableRoutes = map[string]bool{}
	}

	path = r.PathPrefix + path
	if path == "" {
		panic("r2: route path cannot be empty")
	}

	hasTrailingSlash := path[len(path)-1] == '/'
	path = ppath.Clean(path)
	if hasTrailingSlash && path != "/" {
		path += "/"
	}

	if path[0] != '/' {
		panic("r2: route path must start with '/'")
	} else if strings.Count(path, ":") > 1 {
		for _, p := range strings.Split(path, "/") {
			if strings.Count(p, ":") > 1 {
				panic("r2: only one ':' is allowed in a " +
					"route path element")
			}
		}
	} else if strings.Contains(path, "*") {
		if strings.Count(path, "*") > 1 {
			panic("r2: only one '*' is allowed in a route path")
		}

		if path[len(path)-1] != '*' {
			panic("r2: '*' can only appear at the end of a route " +
				"path")
		}

		if strings.Contains(path[strings.LastIndex(path, "/"):], ":") {
			panic("r2: ':' and '*' cannot appear in the same " +
				"route path element")
		}
	}

	routeName := method + path
	for i, l := len(method), len(routeName); i < l; i++ {
		if routeName[i] == ':' {
			j := i + 1
			for ; i < l && routeName[i] != '/'; i++ {
			}

			routeName = routeName[:j] + routeName[i:]

			if i, l = j, len(routeName); i == l {
				break
			}
		}
	}

	if r.registeredRoutes[routeName] {
		if !r.overridableRoutes[routeName] {
			panic("r2: route already exists")
		}

		delete(r.overridableRoutes, routeName)
	} else {
		r.registeredRoutes[routeName] = true
	}

	if h == nil {
		panic("r2: route handler cannot be nil")
	}

	rms := append(r.Middlewares, ms...)
	rh := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		h := h
		for i := len(rms) - 1; i >= 0; i-- {
			h = rms[i].ChainHTTPHandler(h)
		}

		h.ServeHTTP(rw, req)
	})

	var pathParamSlots []*pathParamSlot
	for i, l, slot := 0, len(path), 0; i < l; i++ {
		switch path[i] {
		case '/':
			slot++
		case ':':
			r.insertRoute(
				method,
				path[:i],
				nil,
				staticRouteNode,
				nil,
			)

			offset := i - strings.LastIndexByte(path[:i], '/') - 1

			j := i + 1
			for ; i < l && path[i] != '/'; i++ {
			}

			pathParamSlots = append(pathParamSlots, &pathParamSlot{
				number: slot,
				name:   path[j:i],
				offset: offset,
			})

			path = path[:j] + path[i:]

			if i, l = j, len(path); i == l {
				r.insertRoute(
					method,
					path,
					rh,
					paramRouteNode,
					pathParamSlots,
				)
				return
			}

			r.insertRoute(
				method,
				path[:i],
				nil,
				paramRouteNode,
				pathParamSlots,
			)

			slot++
		case '*':
			r.insertRoute(
				method,
				path[:i],
				nil,
				staticRouteNode,
				nil,
			)

			offset := i - strings.LastIndexByte(path[:i], '/') - 1
			if offset == 0 && i > 1 {
				method := ""
				path := path[:i-1]
				routeName := method + path
				if !r.registeredRoutes[routeName] {
					r.registeredRoutes[routeName] = true
					r.overridableRoutes[routeName] = true
					r.insertRoute(
						method,
						path,
						r.tsrHandler(),
						staticRouteNode,
						nil,
					)
				}
			}

			pathParamSlots = append(pathParamSlots, &pathParamSlot{
				number: slot,
				name:   "*",
				offset: offset,
			})

			r.insertRoute(
				method,
				path[:i+1],
				rh,
				wildcardParamRouteNode,
				pathParamSlots,
			)

			return
		}
	}

	r.insertRoute(method, path, rh, staticRouteNode, pathParamSlots)
}

// insertRoute inserts a new route into the `r.routeTree`.
func (r *Router) insertRoute(
	method string,
	path string,
	h http.Handler,
	nt routeNodeType,
	pathParamSlots []*pathParamSlot,
) {
	var (
		s  = path        // Search
		cn = r.routeTree // Current node
		nn *routeNode    // Next node
		sl int           // Search length
		pl int           // Prefix length
		ll int           // LCP length
		ml int           // Minimum length of the `sl` and `pl`
	)

	for {
		sl, pl, ll = len(s), len(cn.prefix), 0
		if sl < pl {
			ml = sl
		} else {
			ml = pl
		}

		for ; ll < ml && s[ll] == cn.prefix[ll]; ll++ {
		}

		if ll == 0 { // At root node
			cn.prefix = s
			cn.label = s[0]
			cn.typ = nt
			cn.pathParamSlots = pathParamSlots
			if h != nil {
				cn.handlers[method] = h
			}
		} else if ll < pl { // Split node
			nn = &routeNode{
				prefix:         cn.prefix[ll:],
				label:          cn.prefix[ll],
				typ:            cn.typ,
				children:       cn.children,
				pathParamSlots: cn.pathParamSlots,
				handlers:       cn.handlers,
			}

			// Reset current node.
			cn.prefix = cn.prefix[:ll]
			cn.label = cn.prefix[0]
			cn.typ = staticRouteNode
			cn.children = []*routeNode{nn}
			cn.pathParamSlots = nil
			cn.handlers = map[string]http.Handler{}

			if ll == sl { // At current node
				cn.typ = nt
				cn.pathParamSlots = pathParamSlots
				if h != nil {
					cn.handlers[method] = h
				}
			} else { // Create child node
				nn = &routeNode{
					prefix:         s[ll:],
					label:          s[ll],
					typ:            nt,
					pathParamSlots: pathParamSlots,
				}
				if h != nil {
					nn.handlers = map[string]http.Handler{
						method: h,
					}
				} else {
					nn.handlers = map[string]http.Handler{}
				}

				cn.children = append(cn.children, nn)
			}
		} else if ll < sl {
			s = s[ll:]
			if nn = cn.lChild(s[0]); nn != nil {
				// Go deeper.
				cn = nn
				continue
			}

			// Create child node.
			nn = &routeNode{
				prefix:         s,
				label:          s[0],
				typ:            nt,
				pathParamSlots: pathParamSlots,
			}
			if h != nil {
				nn.handlers = map[string]http.Handler{
					method: h,
				}
			} else {
				nn.handlers = map[string]http.Handler{}
			}

			cn.children = append(cn.children, nn)
		} else { // Node already exists
			if len(cn.pathParamSlots) == 0 {
				cn.pathParamSlots = pathParamSlots
			}

			if h != nil {
				cn.handlers[method] = h
			}
		}

		break
	}
}

// Handler returns a matched `http.Handler` for the `req`.
func (r *Router) Handler(req *http.Request) http.Handler {
	if r.Parent != nil {
		return r.Parent.Handler(req)
	}

	if req.RequestURI == "" || req.RequestURI[0] != '/' {
		return r.notFoundHandler()
	}

	path := req.RequestURI
	if i := strings.IndexByte(path, '?'); i > 0 {
		path = path[:i]
	}

	var (
		s     = path        // Search
		cn    = r.routeTree // Current node
		nn    *routeNode    // Next node
		sn    *routeNode    // Saved node
		snt   routeNodeType // Saved node type
		ss    string        // Saved search
		swppn *routeNode    // Saved wildcard param parent node
		swpps string        // Saved wildcard param parent search
		sl    int           // Search length
		pl    int           // Prefix length
		ll    int           // LCP length
		ml    int           // Minimum length of the `sl` and `pl`
		i     int           // Index
	)

	// Search order: static > param > wildcard param.
	for {
		if s == "" {
			if len(cn.handlers) == 0 {
				if cn.tChild(paramRouteNode) != nil {
					goto TryParamNode
				}

				if cn.tChild(wildcardParamRouteNode) != nil {
					goto TryWildcardParamNode
				}

				if swppn != nil {
					goto Struggle
				}
			}

			break
		}

		// Skip continuous "/".
		if s[0] == '/' {
			for i, sl = 1, len(s); i < sl && s[i] == '/'; i++ {
			}

			s = s[i-1:]
		}

		pl, ll = 0, 0
		if cn.label != ':' {
			sl, pl = len(s), len(cn.prefix)
			if sl < pl {
				ml = sl
			} else {
				ml = pl
			}

			for ; ll < ml && s[ll] == cn.prefix[ll]; ll++ {
			}
		}

		if ll != pl {
			goto Struggle
		}

		s = s[ll:]
		if s == "" {
			continue
		}

		// Save wildcard param parent node for struggling.
		if cn != swppn && cn.tChild(wildcardParamRouteNode) != nil {
			swppn = cn
			swpps = s
		}

		// Try static node.
		if nn = cn.child(s[0], staticRouteNode); nn != nil {
			// Save node for struggling.
			pl = len(cn.prefix)
			if pl > 0 && cn.prefix[pl-1] == '/' {
				sn = cn
				snt = paramRouteNode
				ss = s
			}

			cn = nn

			continue
		}

		// Try param node.
	TryParamNode:
		if nn = cn.tChild(paramRouteNode); nn != nil {
			// Save node for struggling.
			pl = len(cn.prefix)
			if pl > 0 && cn.prefix[pl-1] == '/' {
				sn = cn
				snt = wildcardParamRouteNode
				ss = s
			}

			cn = nn

			for i, sl = 0, len(s); i < sl && s[i] != '/'; i++ {
			}

			s = s[i:]

			continue
		}

		// Try wildcard param node.
	TryWildcardParamNode:
		if cn = cn.tChild(wildcardParamRouteNode); cn != nil {
			break
		}

		// Struggle for the former node.
	Struggle:
		if sn != nil {
			cn = sn
			sn = nil
			s = ss
			switch snt {
			case paramRouteNode:
				goto TryParamNode
			case wildcardParamRouteNode:
				goto TryWildcardParamNode
			}
		} else if swppn != nil {
			cn = swppn
			swppn = nil
			s = swpps
			goto TryWildcardParamNode
		}

		return r.notFoundHandler()
	}

	h := cn.handlers[req.Method]
	if h == nil {
		h = cn.handlers[""]
		if h == nil {
			if len(cn.handlers) > 0 {
				return r.methodNotAllowedHandler()
			}

			return r.notFoundHandler()
		}
	}

	return http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		data := map[interface{}]interface{}{
			routeNodeKey: cn,
			pathKey:      path,
		}

		ctx := context.WithValue(req.Context(), dataKey, data)
		h.ServeHTTP(rw, req.WithContext(ctx))

		if dfsi, ok := data[deferredFuncsKey]; ok {
			dfs := dfsi.([]func())
			for i := len(dfs) - 1; i >= 0; i-- {
				dfs[i]()
			}
		}
	})
}

// notFoundHandler returns an `http.Handler` to write 404 not found responses.
func (r *Router) notFoundHandler() http.Handler {
	if r.NotFoundHandler != nil {
		return r.NotFoundHandler
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		http.Error(
			rw,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound,
		)
	})
}

// methodNotAllowedHandler returns an `http.Handler` to write 405 method not
// allowed responses.
func (r *Router) methodNotAllowedHandler() http.Handler {
	if r.MethodNotAllowedHandler != nil {
		return r.MethodNotAllowedHandler
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		http.Error(
			rw,
			http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed,
		)
	})
}

// tsrHandler returns an `http.Handler` to write TSR (trailing slash redirect)
// responses.
func (r *Router) tsrHandler() http.Handler {
	if r.TSRHandler != nil {
		return r.TSRHandler
	}

	return http.HandlerFunc(func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		requestURI := req.RequestURI
		if requestURI == "" {
			requestURI = "/"
		} else if i := strings.IndexByte(requestURI, '?'); i >= 0 {
			path := requestURI[:i]
			if path == "" || path[len(path)-1] != '/' {
				path += "/"
			}

			requestURI = path + requestURI[i:]
		} else if requestURI[len(requestURI)-1] != '/' {
			requestURI += "/"
		}

		http.Redirect(rw, req, requestURI, http.StatusMovedPermanently)
	})
}

// ServeHTTP implements the `http.Handler`.
func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if r.Parent != nil {
		r.Parent.ServeHTTP(rw, req)
		return
	}

	r.Handler(req).ServeHTTP(rw, req)
}

// routeNode is the node of the route radix tree.
type routeNode struct {
	prefix         string
	label          byte
	typ            routeNodeType
	children       []*routeNode
	pathParamSlots []*pathParamSlot
	handlers       map[string]http.Handler
}

// child returns a child node of the `rn` for the `label` and `typ`.
func (rn *routeNode) child(label byte, typ routeNodeType) *routeNode {
	for _, c := range rn.children {
		if c.label == label && c.typ == typ {
			return c
		}
	}

	return nil
}

// lChild returns a child node of the `rn` for the `label`.
func (rn *routeNode) lChild(label byte) *routeNode {
	for _, c := range rn.children {
		if c.label == label {
			return c
		}
	}

	return nil
}

// tChild returns a child node of the `rn` for the `typ`.
func (rn *routeNode) tChild(typ routeNodeType) *routeNode {
	for _, c := range rn.children {
		if c.typ == typ {
			return c
		}
	}

	return nil
}

// routeNodeType is the type of the `routeNode`.
type routeNodeType uint8

// The route node types.
const (
	staticRouteNode routeNodeType = iota
	paramRouteNode
	wildcardParamRouteNode
)

// pathParamSlot is the path parameter slot.
type pathParamSlot struct {
	number int
	name   string
	offset int
}
