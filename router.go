package r2

import (
	"context"
	"net/http"
	stdpath "path"
	"strings"
	"sync"
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

	routeTree           *routeNode
	registeredRoutes    map[string]bool
	overridableRoutes   map[string]bool
	maxPathParams       int
	pathParamValuesPool sync.Pool
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

// Handle registers a new route for the `method` ("" means catch-all) and `path`
// with the matching `h` and optional `ms`.
//
// Note that a ':' followed by a name in the `path` declares a path parameter
// that matches all characters except '/'. And an '*' in the `path` declares a
// wildcard path parameter that greedily matches all characters, with "*" as its
// name. The `PathParam` can be used to get those declared path parameters after
// a request is matched.
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
			methodHandlerSet: &methodHandlerSet{},
		}

		r.registeredRoutes = map[string]bool{}
		r.overridableRoutes = map[string]bool{}
	}

	path = r.PathPrefix + path
	if path == "" {
		panic("r2: route path cannot be empty")
	}

	hasTrailingSlash := path[len(path)-1] == '/'
	path = stdpath.Clean(path)
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
		if i := req.Context().Value(pathParamsContextKey); i != nil {
			r.pathParamValuesPool.Put(i.(*pathParams).values)
		}
	})

	var pathParamNames []string
	for i, l := 0, len(path); i < l; i++ {
		switch path[i] {
		case ':':
			r.insertRoute(
				method,
				path[:i],
				nil,
				staticRouteNode,
				nil,
			)

			j := i + 1
			for ; i < l && path[i] != '/'; i++ {
			}

			pathParamName := path[j:i]
			for _, pn := range pathParamNames {
				if pn == pathParamName {
					panic("r2: route path cannot have " +
						"duplicate parameter names")
				}
			}

			pathParamNames = append(pathParamNames, pathParamName)
			path = path[:j] + path[i:]

			if i, l = j, len(path); i == l {
				r.insertRoute(
					method,
					path[:i],
					rh,
					paramRouteNode,
					pathParamNames,
				)
			} else {
				r.insertRoute(
					method,
					path[:i],
					nil,
					paramRouteNode,
					pathParamNames,
				)
			}
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

			pathParamNames = append(pathParamNames, "*")

			r.insertRoute(
				method,
				path[:i+1],
				rh,
				wildcardParamRouteNode,
				pathParamNames,
			)
		}
	}

	r.insertRoute(method, path, rh, staticRouteNode, pathParamNames)
}

// insertRoute inserts a new route into the `r.routeTree`.
func (r *Router) insertRoute(
	method string,
	path string,
	h http.Handler,
	nt routeNodeType,
	pathParamNames []string,
) {
	if l := len(pathParamNames); r.maxPathParams < l {
		r.maxPathParams = l
		r.pathParamValuesPool = sync.Pool{
			New: func() interface{} {
				return make([]string, l)
			},
		}
	}

	var (
		s  = path        // Search
		sl int           // Search length
		pl int           // Prefix length
		ll int           // LCP length
		ml int           // Minimum length of the `sl` and `pl`
		cn = r.routeTree // Current node
		nn *routeNode    // Next node
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
			if h != nil {
				cn.typ = nt
				cn.pathParamNames = pathParamNames
				cn.setHandler(method, h)
			}
		} else if ll < pl { // Split node
			nn = &routeNode{
				prefix:               cn.prefix[ll:],
				label:                cn.prefix[ll],
				typ:                  cn.typ,
				parent:               cn,
				staticChildren:       cn.staticChildren,
				paramChild:           cn.paramChild,
				wildcardParamChild:   cn.wildcardParamChild,
				hasAtLeastOneChild:   cn.hasAtLeastOneChild,
				pathParamNames:       cn.pathParamNames,
				methodHandlerSet:     cn.methodHandlerSet,
				otherMethodHandlers:  cn.otherMethodHandlers,
				catchAllHandler:      cn.catchAllHandler,
				hasAtLeastOneHandler: cn.hasAtLeastOneHandler,
			}

			for _, n := range nn.staticChildren {
				n.parent = nn
			}

			if nn.paramChild != nil {
				nn.paramChild.parent = nn
			}

			if nn.wildcardParamChild != nil {
				nn.wildcardParamChild.parent = nn
			}

			// Reset current node.
			cn.prefix = cn.prefix[:ll]
			cn.label = cn.prefix[0]
			cn.typ = staticRouteNode
			cn.staticChildren = nil
			cn.paramChild = nil
			cn.wildcardParamChild = nil
			cn.hasAtLeastOneChild = false
			cn.pathParamNames = nil
			cn.methodHandlerSet = &methodHandlerSet{}
			cn.otherMethodHandlers = nil
			cn.catchAllHandler = nil
			cn.hasAtLeastOneHandler = false
			cn.addChild(nn)

			if ll == sl { // At current node
				cn.typ = nt
				cn.pathParamNames = pathParamNames
				cn.setHandler(method, h)
			} else { // Create child node
				nn = &routeNode{
					prefix:           s[ll:],
					label:            s[ll],
					typ:              nt,
					parent:           cn,
					pathParamNames:   pathParamNames,
					methodHandlerSet: &methodHandlerSet{},
				}

				nn.setHandler(method, h)

				cn.addChild(nn)
			}
		} else if ll < sl {
			s = s[ll:]
			switch nn = nil; s[0] {
			case ':':
				nn = cn.paramChild
			case '*':
				nn = cn.wildcardParamChild
			default:
				for _, n := range cn.staticChildren {
					if n.label == s[0] {
						nn = n
						break
					}
				}
			}

			if nn != nil {
				// Go deeper.
				cn = nn
				continue
			}

			// Create child node.
			nn = &routeNode{
				prefix:           s,
				label:            s[0],
				typ:              nt,
				parent:           cn,
				pathParamNames:   pathParamNames,
				methodHandlerSet: &methodHandlerSet{},
			}

			nn.setHandler(method, h)

			cn.addChild(nn)
		} else if h != nil { // Node already exists
			if len(cn.pathParamNames) == 0 {
				cn.pathParamNames = pathParamNames
			}

			cn.setHandler(method, h)
		}

		break
	}
}

// Handler returns a matched `http.Handler` for the `req` alongside with its
// revision.
func (r *Router) Handler(req *http.Request) (http.Handler, *http.Request) {
	if r.Parent != nil {
		return r.Parent.Handler(req)
	}

	if req.RequestURI == "" || req.RequestURI[0] != '/' {
		return r.notFoundHandler(), req
	}

	path := req.RequestURI
	for i := 1; i < len(path); i++ {
		if path[i] == '?' {
			path = path[:i]
			break
		}
	}

	var (
		s    = path        // Search
		si   int           // Search index
		sl   int           // Search length
		pl   int           // Prefix length
		ll   int           // LCP length
		ml   int           // Minimum length of the `sl` and `pl`
		cn   = r.routeTree // Current node
		nn   *routeNode    // Next node
		nnt  routeNodeType // Next node type
		sn   *routeNode    // Saved node
		snt  routeNodeType // Saved node type
		ppi  int           // Path parameter index
		ppvs []string      // Path parameter values
		i    int           // Index
		mh   http.Handler  // Matched `http.Handler`
	)

	// Node search order: static > parameter > wildcard parameter.
OuterLoop:
	for {
		// Skip continuous '/'.
		if s != "" && s[0] == '/' {
			for i, sl = 1, len(s); i < sl && s[i] == '/'; i++ {
			}

			s = s[i-1:]
		}

		pl, ll = 0, 0
		if cn.typ == staticRouteNode {
			sl, pl = len(s), len(cn.prefix)
			if sl < pl {
				ml = sl
			} else {
				ml = pl
			}

			for ; ll < ml && s[ll] == cn.prefix[ll]; ll++ {
			}

			if ll != pl {
				snt = staticRouteNode
				goto StruggleForTheFormerNode
			}

			s = s[ll:]
			si += ll
		}

		if s == "" && cn.hasAtLeastOneHandler {
			if sn == nil {
				sn = cn
			}

			var h http.Handler
			switch req.Method {
			case http.MethodGet:
				h = cn.methodHandlerSet.get
			case http.MethodHead:
				h = cn.methodHandlerSet.head
			case http.MethodPost:
				h = cn.methodHandlerSet.post
			case http.MethodPut:
				h = cn.methodHandlerSet.put
			case http.MethodPatch:
				h = cn.methodHandlerSet.patch
			case http.MethodDelete:
				h = cn.methodHandlerSet.delete
			case http.MethodConnect:
				h = cn.methodHandlerSet.connect
			case http.MethodOptions:
				h = cn.methodHandlerSet.options
			case http.MethodTrace:
				h = cn.methodHandlerSet.trace
			default:
				for _, omh := range cn.otherMethodHandlers {
					if req.Method == omh.method {
						mh = omh.handler
						break OuterLoop
					}
				}
			}

			if h == nil {
				h = cn.catchAllHandler
			}

			if h != nil {
				mh = h
				break
			}
		}

		// Try static node.
		if s != "" {
			for _, n := range cn.staticChildren {
				if n.label == s[0] {
					cn = n
					continue OuterLoop
				}
			}
		}

		// Try parameter node.
	TryParamNode:
		if cn.paramChild != nil {
			cn = cn.paramChild

			for i, sl = 0, len(s); i < sl && s[i] != '/'; i++ {
			}

			if ppvs == nil {
				ppvs = r.pathParamValuesPool.Get().([]string)
			}

			ppvs[ppi] = s[:i]
			ppi++

			s = s[i:]
			si += i

			continue
		}

		// Try wildcard parameter node.
	TryWildcardParamNode:
		if cn.wildcardParamChild != nil {
			cn = cn.wildcardParamChild

			if ppvs == nil {
				ppvs = r.pathParamValuesPool.Get().([]string)
			}

			ppvs[ppi] = s
			ppi++

			si += len(s)
			s = ""

			if sn == nil {
				sn = cn
			}

			var h http.Handler
			switch req.Method {
			case http.MethodGet:
				h = cn.methodHandlerSet.get
			case http.MethodHead:
				h = cn.methodHandlerSet.head
			case http.MethodPost:
				h = cn.methodHandlerSet.post
			case http.MethodPut:
				h = cn.methodHandlerSet.put
			case http.MethodPatch:
				h = cn.methodHandlerSet.patch
			case http.MethodDelete:
				h = cn.methodHandlerSet.delete
			case http.MethodConnect:
				h = cn.methodHandlerSet.connect
			case http.MethodOptions:
				h = cn.methodHandlerSet.options
			case http.MethodTrace:
				h = cn.methodHandlerSet.trace
			default:
				for _, omh := range cn.otherMethodHandlers {
					if req.Method == omh.method {
						mh = omh.handler
						break OuterLoop
					}
				}
			}

			if h == nil {
				h = cn.catchAllHandler
			}

			if h != nil {
				mh = h
				break
			}
		}

		snt = wildcardParamRouteNode

		// Struggle for the former node.
	StruggleForTheFormerNode:
		nn = cn
		if nn.typ < wildcardParamRouteNode {
			nnt = nn.typ + 1
		} else {
			nnt = staticRouteNode
		}

		if snt != staticRouteNode {
			if nn.typ == staticRouteNode {
				si -= len(nn.prefix)
			} else {
				ppi--
				si -= len(ppvs[ppi])
			}

			s = path[si:]
		}

		if cn = cn.parent; cn != nil {
			switch nnt {
			case paramRouteNode:
				goto TryParamNode
			case wildcardParamRouteNode:
				goto TryWildcardParamNode
			}
		} else if snt == staticRouteNode {
			sn = nil
		}

		break
	}

	if cn == nil || mh == nil {
		if ppvs != nil {
			r.pathParamValuesPool.Put(ppvs)
		}

		if sn != nil {
			cn = sn
			if cn.hasAtLeastOneHandler {
				return r.methodNotAllowedHandler(), req
			}
		}

		return r.notFoundHandler(), req
	}

	if len(cn.pathParamNames) > 0 {
		return mh, req.WithContext(context.WithValue(
			req.Context(),
			pathParamsContextKey,
			&pathParams{
				names:  cn.pathParamNames,
				values: ppvs,
			},
		))
	}

	return mh, req
}

// ServeHTTP implements the `http.Handler`.
func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if r.Parent != nil {
		r.Parent.ServeHTTP(rw, req)
		return
	}

	h, req := r.Handler(req)
	h.ServeHTTP(rw, req)
}

// notFoundHandler returns an `http.Handler` to write 404 not found responses.
func (r *Router) notFoundHandler() http.Handler {
	if r.NotFoundHandler != nil {
		return r.NotFoundHandler
	}

	return http.HandlerFunc(notFound)
}

// methodNotAllowedHandler returns an `http.Handler` to write 405 method not
// allowed responses.
func (r *Router) methodNotAllowedHandler() http.Handler {
	if r.MethodNotAllowedHandler != nil {
		return r.MethodNotAllowedHandler
	}

	return http.HandlerFunc(methodNotAllowed)
}

// tsrHandler returns an `http.Handler` to write TSR (trailing slash redirect)
// responses.
func (r *Router) tsrHandler() http.Handler {
	if r.TSRHandler != nil {
		return r.TSRHandler
	}

	return http.HandlerFunc(tsr)
}

// notFound writes a 404 not found response.
func notFound(rw http.ResponseWriter, req *http.Request) {
	http.Error(
		rw,
		http.StatusText(http.StatusNotFound),
		http.StatusNotFound,
	)
}

// methodNotAllowed writes a 405 method not allowed response.
func methodNotAllowed(rw http.ResponseWriter, req *http.Request) {
	http.Error(
		rw,
		http.StatusText(http.StatusMethodNotAllowed),
		http.StatusMethodNotAllowed,
	)
}

// tsr writes a TSR (trailing slash redirect) response.
func tsr(rw http.ResponseWriter, req *http.Request) {
	requestURI := req.RequestURI
	if requestURI == "" {
		requestURI = "/"
	} else {
		path, query := requestURI, ""
		for i := 0; i < len(path); i++ {
			if path[i] == '?' {
				query = path[i:]
				path = path[:i]
				break
			}
		}

		if path == "" || path[len(path)-1] != '/' {
			path += "/"
		}

		requestURI = path + query
	}

	http.Redirect(rw, req, requestURI, http.StatusMovedPermanently)
}

// routeNode is the node of the route radix tree.
type routeNode struct {
	prefix               string
	label                byte
	typ                  routeNodeType
	parent               *routeNode
	staticChildren       []*routeNode
	paramChild           *routeNode
	wildcardParamChild   *routeNode
	hasAtLeastOneChild   bool
	pathParamNames       []string
	methodHandlerSet     *methodHandlerSet
	otherMethodHandlers  []*methodHandler
	catchAllHandler      http.Handler
	hasAtLeastOneHandler bool
}

// addChild adds the `n` as a child node to the `rn`.
func (rn *routeNode) addChild(n *routeNode) {
	switch n.typ {
	case staticRouteNode:
		rn.staticChildren = append(rn.staticChildren, n)
	case paramRouteNode:
		rn.paramChild = n
	case wildcardParamRouteNode:
		rn.wildcardParamChild = n
	}

	rn.hasAtLeastOneChild = true
}

// setHandler sets the `h` to the `rn` based on the `method`.
func (rn *routeNode) setHandler(method string, h http.Handler) {
	switch method {
	case "":
		rn.catchAllHandler = h
	case http.MethodGet:
		rn.methodHandlerSet.get = h
	case http.MethodHead:
		rn.methodHandlerSet.head = h
	case http.MethodPost:
		rn.methodHandlerSet.post = h
	case http.MethodPut:
		rn.methodHandlerSet.put = h
	case http.MethodPatch:
		rn.methodHandlerSet.patch = h
	case http.MethodDelete:
		rn.methodHandlerSet.delete = h
	case http.MethodConnect:
		rn.methodHandlerSet.connect = h
	case http.MethodOptions:
		rn.methodHandlerSet.options = h
	case http.MethodTrace:
		rn.methodHandlerSet.trace = h
	default:
		var exists bool
		for i, mh := range rn.otherMethodHandlers {
			if method == mh.method {
				if h != nil {
					mh.handler = h
				} else {
					rn.otherMethodHandlers = append(
						rn.otherMethodHandlers[:i],
						rn.otherMethodHandlers[i+1:]...,
					)
				}

				exists = true

				break
			}
		}

		if !exists && h != nil {
			rn.otherMethodHandlers = append(
				rn.otherMethodHandlers,
				&methodHandler{
					method:  method,
					handler: h,
				},
			)
		}
	}

	if h != nil {
		rn.hasAtLeastOneHandler = true
	} else {
		rn.hasAtLeastOneHandler = rn.methodHandlerSet.get != nil ||
			rn.methodHandlerSet.head != nil ||
			rn.methodHandlerSet.post != nil ||
			rn.methodHandlerSet.put != nil ||
			rn.methodHandlerSet.patch != nil ||
			rn.methodHandlerSet.delete != nil ||
			rn.methodHandlerSet.connect != nil ||
			rn.methodHandlerSet.options != nil ||
			rn.methodHandlerSet.trace != nil ||
			len(rn.otherMethodHandlers) > 0 ||
			rn.catchAllHandler != nil
	}
}

// routeNodeType is the type of the `routeNode`.
type routeNodeType uint8

// The route node types.
const (
	staticRouteNode routeNodeType = iota
	paramRouteNode
	wildcardParamRouteNode
)

// methodHandlerSet is a set of `http.Handler`s for the well-known HTTP methods.
type methodHandlerSet struct {
	get     http.Handler
	head    http.Handler
	post    http.Handler
	put     http.Handler
	patch   http.Handler
	delete  http.Handler
	connect http.Handler
	options http.Handler
	trace   http.Handler
}

// methodHandler is a `http.Handler` for an HTTP method.
type methodHandler struct {
	method  string
	handler http.Handler
}

// pathParams is the path parameters.
type pathParams struct {
	names  []string
	values []string
}
