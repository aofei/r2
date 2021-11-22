package r2

import (
	"context"
	"net/http"
	stdpath "path"
	"strings"
	"sync"
)

// Router is a registry of all registered routes for HTTP request routing.
//
// Make sure that all fields of the `Router` have been finalized before calling
// any of its methods.
type Router struct {
	// Parent is the parent `Router`.
	Parent *Router

	// PathPrefix is the path prefix of all routes to be registered.
	PathPrefix string

	// Middlewares is the `Middleware` chain that performs after routing.
	Middlewares []Middleware

	// NotFoundHandler writes not found responses. It is used when the
	// `Handler` fails to find a matching handler for a request.
	//
	// If the `NotFoundHandler` is nil, a default one is used.
	//
	// Note that the `NotFoundHandler` will be ignored when the `Parent` is
	// not nil.
	NotFoundHandler http.Handler

	// MethodNotAllowedHandler writes method not allowed responses. It is
	// used when the `Handler` finds a handler that matches only the path
	// but not the method for a request.
	//
	// If the `MethodNotAllowedHandler` is nil, a default one is used.
	//
	// Note that the `MethodNotAllowedHandler` will be ignored when the
	// `Parent` is not nil.
	MethodNotAllowedHandler http.Handler

	// TSRHandler writes TSR (Trailing Slash Redirect) responses. It may be
	// used when the path of a registered route ends with "/*", and the
	// `Handler` fails to find a matching handler for a request whose path
	// does not end with such pattern.
	//
	// If the `TSRHandler` is nil, a default one is used.
	//
	// Note that the `TSRHandler` will be ignored when the `Parent` is not
	// nil.
	TSRHandler http.Handler

	routeTree                      *routeNode
	registeredRoutes               map[string]bool
	maxPathParams                  int
	pathParamValuesPool            sync.Pool
	chainedNotFoundHandler         http.Handler
	chainedMethodNotAllowedHandler http.Handler
	chainedTSRHandler              http.Handler
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

// Handle registers a new route for the `method` (empty string means catch-all)
// and `path` with the matching `h` and optional `ms`.
//
// A ':' followed by a name in the `path` declares a path parameter that matches
// all characters except '/'. And an '*' in the `path` declares a wildcard path
// parameter that greedily matches all characters, with "*" as its name. The
// `PathParam` can be used to get those declared path parameters after a request
// is matched.
//
// When the `path` ends with "/*", and there is at least one path element before
// it without any other path parameters, a sepcial catch-all route will be
// automatically registered with the result of `path[:len(path)-2]` as its path
// and the `r.TSRHandler` as its handler. This special catch-all route will be
// overridden if a route with such path is explicitly registered, regardless of
// its method.
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
		r.notFoundHandler()
		r.methodNotAllowedHandler()
		r.tsrHandler()
	}

	for _, c := range method {
		if (c < '0' || c > '9') &&
			(c < 'A' || c > 'Z') &&
			(c < 'a' || c > 'z') {
			panic("r2: route method must be alphanumeric")
		}
	}

	path = r.PathPrefix + path
	if path == "" {
		panic("r2: route path cannot be empty")
	} else if path[0] != '/' {
		panic("r2: route path must start with '/'")
	}

	hasTrailingSlash := path[len(path)-1] == '/'
	path = stdpath.Clean(path)
	if hasTrailingSlash && path != "/" {
		path += "/"
	}

	var hasAtLeastOnePathParam bool
	if strings.Contains(path, ":") {
		hasAtLeastOnePathParam = true
		for _, p := range strings.Split(path, "/") {
			if strings.Count(p, ":") > 1 {
				panic("r2: only one ':' is allowed in a " +
					"route path element")
			}
		}
	}

	if strings.Contains(path, "*") {
		hasAtLeastOnePathParam = true

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

			i, l = j, len(routeName)
		}
	}

	if r.registeredRoutes[routeName] {
		panic("r2: route already exists")
	} else {
		r.registeredRoutes[routeName] = true
	}

	if h == nil {
		panic("r2: route handler cannot be nil")
	}

	if ms := append(r.Middlewares, ms...); len(ms) > 0 {
		for i := len(ms) - 1; i >= 0; i-- {
			if ms[i] != nil {
				h = ms[i].ChainHTTPHandler(h)
			}
		}
	}

	if hasAtLeastOnePathParam {
		ph := h
		h = http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			ph.ServeHTTP(rw, req)
			d, ok := req.Context().Value(dataContextKey).(*data)
			if ok {
				r.pathParamValuesPool.Put(d.pathParamValues)
			}
		})
	}

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
			if pathParamName == "" {
				panic("r2: route path parameter name cannot " +
					"be empty")
			}

			for _, pn := range pathParamNames {
				if pn == pathParamName {
					panic("r2: route path cannot have " +
						"duplicate parameter names")
				}
			}

			pathParamNames = append(pathParamNames, pathParamName)
			path = path[:j] + path[i:]

			i, l = j, len(path)
			if i < l {
				r.insertRoute(
					method,
					path[:i],
					nil,
					paramRouteNode,
					pathParamNames,
				)
			} else {
				r.insertRoute(
					method,
					path[:i],
					h,
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
			if offset == 0 && i > 1 && len(pathParamNames) == 0 {
				method, path := "_tsr", path[:i-1]
				routeName := method + path
				if !r.registeredRoutes[routeName] {
					r.registeredRoutes[routeName] = true
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
				h,
				wildcardParamRouteNode,
				pathParamNames,
			)
		}
	}

	r.insertRoute(method, path, h, staticRouteNode, pathParamNames)
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

			nn = nil
			switch s[0] {
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

// Handler returns a matched `http.Handler` for the `req` along with a possible
// revision of the `req`.
//
// The returned `http.Handler` is always non-nil.
//
// The revision of the `req` only happens when the matched route has at least
// one path parameter and the result of `req.Context()` has nothing to do with
// the `Context`. Otherwise, the `req` itself is returned.
func (r *Router) Handler(req *http.Request) (http.Handler, *http.Request) {
	if r.Parent != nil {
		return r.Parent.Handler(req)
	}

	if r.routeTree == nil {
		return r.notFoundHandler(), req
	}

	var (
		s    = req.URL.Path // Search
		si   int            // Search index
		sl   int            // Search length
		pl   int            // Prefix length
		ll   int            // LCP length
		ml   int            // Minimum length of the `sl` and `pl`
		cn   = r.routeTree  // Current node
		sn   *routeNode     // Saved node
		fnt  routeNodeType  // From node type
		nnt  routeNodeType  // Next node type
		ppi  int            // Path parameter index
		ppvs []string       // Path parameter values
		i    int            // Index
		h    http.Handler   // Handler
	)

	// Node search order: static > parameter > wildcard parameter.
OuterLoop:
	for {
		if cn.typ == staticRouteNode {
			sl, pl = len(s), len(cn.prefix)
			if sl < pl {
				ml = sl
			} else {
				ml = pl
			}

			ll = 0
			for ; ll < ml && s[ll] == cn.prefix[ll]; ll++ {
			}

			if ll != pl {
				fnt = staticRouteNode
				goto BacktrackToPreviousNode
			}

			s = s[ll:]
			si += ll
		}

		if s == "" && cn.hasAtLeastOneHandler {
			if sn == nil {
				sn = cn
			}

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
					if omh.method == req.Method {
						h = omh.handler
						break OuterLoop
					}
				}
			}

			if h == nil && cn.catchAllHandler != nil {
				h = cn.catchAllHandler.handler
			}

			if h != nil {
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

			i, sl = 0, len(s)
			for ; i < sl && s[i] != '/'; i++ {
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
					if omh.method == req.Method {
						h = omh.handler
						break OuterLoop
					}
				}
			}

			if h == nil && cn.catchAllHandler != nil {
				h = cn.catchAllHandler.handler
			}

			if h != nil {
				break
			}
		}

		fnt = wildcardParamRouteNode

		// Backtrack to previous node.
	BacktrackToPreviousNode:
		if fnt != staticRouteNode {
			if cn.typ == staticRouteNode {
				si -= len(cn.prefix)
			} else {
				ppi--
				si -= len(ppvs[ppi])
			}

			s = req.URL.Path[si:]
		}

		if cn.typ < wildcardParamRouteNode {
			nnt = cn.typ + 1
		} else {
			nnt = staticRouteNode
		}

		cn = cn.parent
		if cn != nil {
			switch nnt {
			case paramRouteNode:
				goto TryParamNode
			case wildcardParamRouteNode:
				goto TryWildcardParamNode
			}
		} else if fnt == staticRouteNode {
			sn = nil
		}

		break
	}

	if cn == nil || h == nil {
		if ppvs != nil {
			r.pathParamValuesPool.Put(ppvs)
		}

		if sn != nil && sn.hasAtLeastOneHandler {
			return r.methodNotAllowedHandler(), req
		}

		return r.notFoundHandler(), req
	}

	if len(cn.pathParamNames) > 0 {
		if d, ok := req.Context().Value(dataContextKey).(*data); ok {
			d.pathParamNames = cn.pathParamNames
			d.pathParamValues = ppvs
		} else {
			req = req.WithContext(context.WithValue(
				req.Context(),
				dataContextKey,
				&data{
					pathParamNames:  cn.pathParamNames,
					pathParamValues: ppvs,
				},
			))
		}
	}

	return h, req
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

// notFoundHandler returns an `http.Handler` to write not found responses.
func (r *Router) notFoundHandler() http.Handler {
	if r.chainedNotFoundHandler != nil {
		return r.chainedNotFoundHandler
	}

	h := r.NotFoundHandler
	if h == nil {
		h = http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			http.Error(
				rw,
				http.StatusText(http.StatusNotFound),
				http.StatusNotFound,
			)
		})
	}

	if len(r.Middlewares) > 0 {
		for i := len(r.Middlewares) - 1; i >= 0; i-- {
			if r.Middlewares[i] != nil {
				h = r.Middlewares[i].ChainHTTPHandler(h)
			}
		}
	}

	r.chainedNotFoundHandler = h

	return h
}

// methodNotAllowedHandler returns an `http.Handler` to write method not allowed
// responses.
func (r *Router) methodNotAllowedHandler() http.Handler {
	if r.chainedMethodNotAllowedHandler != nil {
		return r.chainedMethodNotAllowedHandler
	}

	h := r.MethodNotAllowedHandler
	if h == nil {
		h = http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
			http.Error(
				rw,
				http.StatusText(http.StatusMethodNotAllowed),
				http.StatusMethodNotAllowed,
			)
		})
	}

	if len(r.Middlewares) > 0 {
		for i := len(r.Middlewares) - 1; i >= 0; i-- {
			if r.Middlewares[i] != nil {
				h = r.Middlewares[i].ChainHTTPHandler(h)
			}
		}
	}

	r.chainedMethodNotAllowedHandler = h

	return h
}

// tsrHandler returns an `http.Handler` to write TSR (Trailing Slash Redirect)
// responses.
func (r *Router) tsrHandler() http.Handler {
	if r.chainedTSRHandler != nil {
		return r.chainedTSRHandler
	}

	h := r.TSRHandler
	if h == nil {
		h = http.HandlerFunc(func(
			rw http.ResponseWriter,
			req *http.Request,
		) {
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

			http.Redirect(
				rw,
				req,
				requestURI,
				http.StatusMovedPermanently,
			)
		})
	}

	if len(r.Middlewares) > 0 {
		for i := len(r.Middlewares) - 1; i >= 0; i-- {
			if r.Middlewares[i] != nil {
				h = r.Middlewares[i].ChainHTTPHandler(h)
			}
		}
	}

	r.chainedTSRHandler = h

	return h
}

// routeNode is a node of a route radix tree.
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
	catchAllHandler      *methodHandler
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
	case "", "_tsr":
		if method == "_tsr" && rn.hasAtLeastOneHandler {
			return
		}

		rn.catchAllHandler = &methodHandler{
			method:  method,
			handler: h,
		}
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
			if mh.method == method {
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

	hasAtLeastOneMethodHandler := rn.methodHandlerSet.get != nil ||
		rn.methodHandlerSet.head != nil ||
		rn.methodHandlerSet.post != nil ||
		rn.methodHandlerSet.put != nil ||
		rn.methodHandlerSet.patch != nil ||
		rn.methodHandlerSet.delete != nil ||
		rn.methodHandlerSet.connect != nil ||
		rn.methodHandlerSet.options != nil ||
		rn.methodHandlerSet.trace != nil ||
		len(rn.otherMethodHandlers) > 0

	if method != "_tsr" &&
		hasAtLeastOneMethodHandler &&
		rn.catchAllHandler != nil &&
		rn.catchAllHandler.method == "_tsr" {
		rn.catchAllHandler = nil
	}

	rn.hasAtLeastOneHandler = hasAtLeastOneMethodHandler ||
		rn.catchAllHandler != nil
}

// routeNodeType is a type of a `routeNode`.
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
