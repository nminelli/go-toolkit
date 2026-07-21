package httprouter

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Config allows configuring a Router instance.
type Config struct {
	ErrorHandlerFunc            ErrorHandlerFunc
	NotFoundHandler             http.Handler
	HealthCheckLivenessHandler  http.Handler
	HealthCheckReadinessHandler http.Handler
	EnableProfiler              bool

	Middlewares []func(http.Handler) http.Handler
}

// WithErrorHandlerFunc allows you to configure the ErrorHandlerFunc for use to
// router.
func WithErrorHandlerFunc(errorHandlerFunc ErrorHandlerFunc) func(options *Config) {
	return func(opt *Config) {
		opt.ErrorHandlerFunc = errorHandlerFunc
	}
}

// WithNotFoundHandler allows you to configure the NotFoundHandler for use to
// router.
func WithNotFoundHandler(notFoundHandler http.Handler) func(options *Config) {
	return func(opt *Config) {
		opt.NotFoundHandler = notFoundHandler
	}
}

// WithHealthCheckLivenessHandler allows you to configure the
// HealthCheckLivenessHandler for use to router.
func WithHealthCheckLivenessHandler(livenessHandler http.Handler) func(options *Config) {
	return func(opt *Config) {
		opt.HealthCheckLivenessHandler = livenessHandler
	}
}

// WithHealthCheckReadinessHandler allows you to configure the
// HealthCheckReadinessHandler for use to router.
func WithHealthCheckReadinessHandler(readinessHandler http.Handler) func(options *Config) {
	return func(opt *Config) {
		opt.HealthCheckReadinessHandler = readinessHandler
	}
}

// WithEnableProfiler allows you to configure the
// EnableProfiler for router.
func WithEnableProfiler(enableProfiler bool) func(options *Config) {
	return func(opt *Config) {
		opt.EnableProfiler = enableProfiler
	}
}

// WithGlobalMiddlewares allows you to configure the
// Middlewares for use to router.
func WithGlobalMiddlewares(middlewares ...func(http.Handler) http.Handler) func(options *Config) {
	return func(opt *Config) {
		opt.Middlewares = middlewares
	}
}

// Router is a http.Handler which can be used to dispatch requests to different
// handler functions via configurable routes.
type Router struct {
	mux            chi.Router
	errHandlerFunc ErrorHandlerFunc
	mountedRouters map[string]*Router // Track mounted subrouters by their mount pattern
}

// New instantiates a `Router` with the given configuration.
func New(optFns ...func(options *Config)) *Router {
	mux := chi.NewRouter()

	var opts Config
	for _, fn := range optFns {
		fn(&opts)
	}

	if opts.Middlewares != nil {
		mux.Use(opts.Middlewares...)
	}

	if opts.NotFoundHandler != nil {
		mux.NotFound(opts.NotFoundHandler.ServeHTTP)
	}

	if opts.HealthCheckLivenessHandler != nil {
		mux.Get("/health/live", opts.HealthCheckLivenessHandler.ServeHTTP)
	}

	if opts.HealthCheckReadinessHandler != nil {
		mux.Get("/health/ready", opts.HealthCheckReadinessHandler.ServeHTTP)
	}

	if opts.EnableProfiler {
		mux.Mount("/debug", middleware.Profiler())
	}

	return &Router{
		mux:            mux,
		errHandlerFunc: opts.ErrorHandlerFunc,
		mountedRouters: make(map[string]*Router),
	}
}

// Use appends a middleware handler to the Mux middleware stack.
//
// The middleware stack for any Mux will execute before searching for a matching
// route to a specific handler, which provides opportunity to respond early,
// change the course of the request execution, or set request-scoped values for
// the next http.Handler.
func (r *Router) Use(middleware ...func(http.Handler) http.Handler) {
	r.mux.Use(middleware...)
}

// With adds inline middlewares for an endpoint handler.
func (r *Router) With(middlewares ...func(http.Handler) http.Handler) *Router {
	return &Router{
		mux:            r.mux.With(middlewares...),
		errHandlerFunc: r.errHandlerFunc,
		mountedRouters: make(map[string]*Router), // New router doesn't inherit mounted routers
	}
}

// Group creates a new inline-Mux with a copy of middleware stack. It's useful
// for a group of handlers along the same routing path that use an additional
// set of middlewares.
func (r *Router) Group(fn func(r Router)) *Router {
	im := r.With()
	if fn != nil {
		fn(*im)
	}

	return im
}

// Route creates a new Mux and mounts it along the `pattern` as a subrouter.
// Effectively, this is a shorthand call to Mount.
func (r *Router) Route(pattern string, fn func(r Router)) *Router {
	if fn == nil {
		panic(fmt.Sprintf("httprouter: attempting to Route() a nil subrouter on '%s'", pattern))
	}

	subRouter := New()
	fn(*subRouter)
	r.mux.Mount(pattern, subRouter)

	// Store the subrouter in our registry for route discovery
	r.mountedRouters[pattern] = subRouter

	return subRouter
}

// Mount attaches another http.Handler or chi Router as a subrouter along a routing
// path. It's very useful to split up a large API as many independent routers and
// compose them as a single service using Mount.
func (r *Router) Mount(pattern string, handler http.Handler) {
	r.mux.Mount(pattern, handler)
}

// Get adds the route `pattern` that matches a GET http method to
// execute the `handlerFn` http.HandlerFunc.
func (r *Router) Get(pattern string, handler http.Handler) {
	r.mux.Get(pattern, handler.ServeHTTP)
}

// Delete adds the route `pattern` that matches a DELETE http method to
// execute the `handlerFn` http.HandlerFunc.
func (r *Router) Delete(pattern string, handler http.Handler) {
	r.mux.Delete(pattern, handler.ServeHTTP)
}

// Head adds the route `pattern` that matches a HEAD http method to
// execute the `handlerFn` http.HandlerFunc.
func (r *Router) Head(pattern string, handler http.Handler) {
	r.mux.Head(pattern, handler.ServeHTTP)
}

// Options adds the route `pattern` that matches a OPTIONS http method to
// execute the `handlerFn` http.HandlerFunc.
func (r *Router) Options(pattern string, handler http.Handler) {
	r.mux.Options(pattern, handler.ServeHTTP)
}

// Patch adds the route `pattern` that matches a PATCH http method to
// execute the `handlerFn` http.HandlerFunc.
func (r *Router) Patch(pattern string, handler http.Handler) {
	r.mux.Patch(pattern, handler.ServeHTTP)
}

// Post adds the route `pattern` that matches a Post http method to
// execute the `handlerFn` http.HandlerFunc.
func (r *Router) Post(pattern string, handler http.Handler) {
	r.mux.Post(pattern, handler.ServeHTTP)
}

// Put adds the route `pattern` that matches a PUT http method to
// execute the `handlerFn` http.HandlerFunc.
func (r *Router) Put(pattern string, handler http.Handler) {
	r.mux.Put(pattern, handler.ServeHTTP)
}

// Trace adds the route `pattern` that matches a TRACE http method to
// execute the `handlerFn` http.HandlerFunc.
func (r *Router) Trace(pattern string, handler http.Handler) {
	r.mux.Trace(pattern, handler.ServeHTTP)
}

// Connect adds the route `pattern` that matches a CONNECT http method to
// execute the `handlerFn` http.HandlerFunc.
func (r *Router) Connect(pattern string, handler http.Handler) {
	r.mux.Connect(pattern, handler.ServeHTTP)
}

// ServeHTTP conforms to the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Route describes the details of a routing handler.
type Route struct {
	Method      string
	Route       string
	Handler     http.Handler
	Middlewares []func(http.Handler) http.Handler
}

// Routes returns the routing tree in an easily traversable structure.
func (r *Router) Routes() ([]Route, error) {
	var routes []Route
	processedMounts := make(map[string]bool) // Track which mount patterns we've already processed

	walkFunc := func(method string, route string, handler http.Handler, mw ...func(http.Handler) http.Handler) error {
		// Check if this is a mounted subrouter pattern (ends with /*)
		if strings.HasSuffix(route, "/*") {
			mountPattern := strings.TrimSuffix(route, "/*")

			// Check if we have a registered subrouter for this mount pattern
			if subRouter, exists := r.mountedRouters[mountPattern]; exists {
				// Only process this mount pattern once to avoid duplicates
				if processedMounts[mountPattern] {
					return nil
				}
				processedMounts[mountPattern] = true

				// Recursively get routes from the subrouter
				subRoutes, err := subRouter.Routes()
				if err != nil {
					return err
				}

				// Prepend the mount pattern to each subroute
				for _, subRoute := range subRoutes {
					fullRoute := mountPattern + subRoute.Route
					routes = append(routes, Route{
						Method:      subRoute.Method,
						Route:       fullRoute,
						Handler:     subRoute.Handler,
						Middlewares: append(mw, subRoute.Middlewares...),
					})
				}
				return nil
			}
		}

		// Regular route, add it as-is
		routes = append(routes, Route{
			Method:      method,
			Route:       route,
			Handler:     handler,
			Middlewares: mw,
		})
		return nil
	}

	if err := chi.Walk(r.mux, walkFunc); err != nil {
		return nil, fmt.Errorf("generating routes %w", err)
	}

	return routes, nil
}
