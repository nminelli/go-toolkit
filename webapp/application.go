package webapp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.23.0"
	"go.opentelemetry.io/otel/semconv/v1.37.0/httpconv"

	"github.com/MFN-AISystems/go-toolkit/httprouter"
	"github.com/MFN-AISystems/go-toolkit/telemetry"
	"github.com/MFN-AISystems/go-toolkit/telemetry/log"
)

const (
	_defaultWebApplicationPort = "8080"

	// Default compression level for defined response content types.
	// The level should be one of the ones defined in the flat package.
	// Higher levels typically run slower but compress more.
	_defaultCompressionLevel = 5
)

var (
	_defaultApplicationName = "Go Web Application"
	_telemetryIgnoreRoutes  = []string{"/health/live", "/health/ready"}
)

// Application is a container struct that contains all required base components
// for building web applications.
type Application struct {
	config            AppOptions
	shutdownTelemetry func()
	shutdownHooks     []func()

	Name   string
	Router *httprouter.Router
	Logger log.Logger
}

// AppOptions represents the options for configuring a web application.
type AppOptions struct {
	ServerTimeouts httprouter.Timeouts
	Listener       net.Listener
	Environment    string
	ErrorHandler   httprouter.ErrorHandlerFunc
	Middlewares    []func(http.Handler) http.Handler
	Logger         log.Logger
	ShutdownHooks  []func()
	LoggingConfig  loggingConfig
}

type loggingConfig struct {
	LogRequests  bool
	LogResponses bool
}

// WithTimeouts allows you to configure the different timeouts
// that the http server uses.
//
// Default behavior is to not have timeouts for incoming requests.
func WithTimeouts(timeout httprouter.Timeouts) func(options *AppOptions) {
	return func(opts *AppOptions) {
		opts.ServerTimeouts = timeout
	}
}

// WithErrorHandler allows you to set a custom error handling function.
//
// The function gets called everytime one of your handlers returns en non-nil error.
// Default is to treat all errors that are not httprouter.Error as 500 status code errors.
func WithErrorHandler(errHandlerFunc httprouter.ErrorHandlerFunc) func(options *AppOptions) {
	return func(opts *AppOptions) {
		opts.ErrorHandler = errHandlerFunc
	}
}

// WithListener allows you to configure the network listener at which the web
// server will be listening to incoming connections.
//
// Default behavior is to use whatever value is in PORT env variable, and if
// none is found, then use 8080.
func WithListener(listener net.Listener) func(options *AppOptions) {
	return func(opts *AppOptions) {
		opts.Listener = listener
	}
}

// WithGlobalMiddlewares allows you to configure the global middlewares to use
// for httprouter.Router
func WithGlobalMiddlewares(middlewares ...func(handler http.Handler) http.Handler) func(options *AppOptions) {
	return func(opts *AppOptions) {
		opts.Middlewares = middlewares
	}
}

// WithShutdownHooks allows you to configure shutdown hooks that will be called
// during graceful shutdown. These hooks are executed before the HTTP server shutdown.
// Example: passing SQS poller Stop() functions to gracefully stop message processing.
func WithShutdownHooks(hooks ...func()) func(options *AppOptions) {
	return func(opts *AppOptions) {
		opts.ShutdownHooks = hooks
	}
}

// WithShutdownHook allows you to add/configure a single shutdown hook that will be called
// during graceful shutdown. This is a convenience function for adding a single hook
// to the existing shutdown hooks in the AppOptions.
func WithShutdownHook(hooks ...func()) func(options *AppOptions) {
	return func(opts *AppOptions) {
		if opts.ShutdownHooks == nil {
			opts.ShutdownHooks = make([]func(), 0)
		}
		opts.ShutdownHooks = append(opts.ShutdownHooks, hooks...)
	}
}

// WithRequestLogging allows you to configure whether the application should log incoming
// HTTP requests body.
func WithRequestLogging(logRequests bool) func(options *AppOptions) {
	return func(opts *AppOptions) {
		opts.LoggingConfig.LogRequests = logRequests
	}
}

// WithResponseLogging allows you to configure whether the application should log outgoing
// HTTP responses body.
func WithResponseLogging(logResponses bool) func(options *AppOptions) {
	return func(opts *AppOptions) {
		opts.LoggingConfig.LogResponses = logResponses
	}
}

// AddShutdownHook adds a shutdown hook that will be called during graceful shutdown.
// This can be used to register shutdown functions after the application is created.
func (a *Application) AddShutdownHook(hook func()) {
	a.shutdownHooks = append(a.shutdownHooks, hook)
}

// Run starts your Application, it blocks until os.Interrupt is received.
func (a *Application) Run() error {
	ctx := context.Background()
	err := a.configureListener()
	if err != nil {
		return err
	}

	defer log.Info(ctx, "Application gracefully shut down")

	a.printBanner()

	if err := a.printRoutes(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Info(ctx, fmt.Sprintf("Application %s successfully started on %s", a.Name, a.config.Listener.Addr().String()))

	// Combine all shutdown hooks from config and registered ones
	allShutdownHooks := append(a.config.ShutdownHooks, a.shutdownHooks...)

	// Add telemetry shutdown hook
	if a.shutdownTelemetry != nil {
		allShutdownHooks = append(allShutdownHooks, a.shutdownTelemetry)
	}

	// Run blocks until the web backend was signaled to close
	if err := httprouter.RunWithShutdownHooks(a.config.Listener, a.config.ServerTimeouts, a.Router, allShutdownHooks...); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error(ctx, "Application startup failed", log.String("error_msg", err.Error()))
		return err
	}

	return nil
}

func (a *Application) configureListener() error {
	if a.config.Listener != nil {
		return nil
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = _defaultWebApplicationPort
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	a.config.Listener = ln

	return nil
}

// printRoutes prints every route grouped by URL and http methods.
// Example:
//
// /path                  [GET POST]
// /path/sub-path         [GET]
// /path/{id}             [POST]
// /ping                  [GET].
func (a *Application) printRoutes() error {
	var w tabwriter.Writer
	w.Init(os.Stdout, 0, 0, 0, ' ', tabwriter.TabIndent)

	routes, err := a.Router.Routes()
	if err != nil {
		return err
	}

	m := make(map[string][]string)
	var r []string
	for _, route := range routes {
		r = append(r, route.Route)
		m[route.Route] = append(m[route.Route], route.Method)
	}

	visited := make(map[string]struct{})
	sort.Strings(r)

	fmt.Println("Registered routes:")
	for _, v := range r {
		if _, ok := visited[v]; !ok {
			sort.Strings(m[v])
			fmt.Fprintf(&w, " - %s\t %v\t\n", v, m[v])
			visited[v] = struct{}{}
		}
	}

	// Flush routes buffer
	err = w.Flush()
	fmt.Println()

	return err
}

// New instantiates a backend Application with sane defaults.
func New(optFns ...func(opts *AppOptions)) (*Application, error) {
	ctx := context.Background()
	var config AppOptions
	for _, fn := range optFns {
		fn(&config)
	}

	appName := os.Getenv("OTEL_SERVICE_NAME")
	if appName == "" {
		appName = _defaultApplicationName
	}

	if config.ServerTimeouts == (httprouter.Timeouts{}) {
		config.ServerTimeouts = httprouter.Timeouts{
			ShutdownTimeout: 5 * time.Second,
		}
	}

	err := LoadEnvFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load environment file: %w", err)
	}
	cancel, err := telemetry.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize telemetry: %w", err)
	}

	logger := configureLogger()
	router := defaultHTTPRouter(appName, logger, config.ErrorHandler, config.LoggingConfig, config.Middlewares...)

	return &Application{
		config:            config,
		shutdownTelemetry: cancel,
		shutdownHooks:     config.ShutdownHooks,
		Name:              appName,
		Router:            router,
		Logger:            logger,
	}, nil
}

func configureLogger() log.Logger {
	return log.NewLogger()
}

func defaultHTTPRouter(
	appName string,
	logger log.Logger,
	errorHandlerFunc httprouter.ErrorHandlerFunc,
	loggingConfig loggingConfig,
	middlewares ...func(handler http.Handler) http.Handler,
) *httprouter.Router {
	middlewares = append(middlewares, []func(http.Handler) http.Handler{
		headerForwarder,
		telemetryMiddleware(appName, logger),
		newCompressor(),
		logMiddleware(loggingConfig),
	}...)

	notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := httprouter.NewErrorf(http.StatusNotFound, "resource %s not found", r.URL.Path)
		JSON(r.Context(), w, http.StatusNotFound, err)
	})

	livenessHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		JSON(r.Context(), w, http.StatusNoContent, nil)
	})

	readinessHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		JSON(r.Context(), w, http.StatusNoContent, nil)
	})

	return httprouter.New(
		httprouter.WithGlobalMiddlewares(middlewares...),
		httprouter.WithNotFoundHandler(notFoundHandler),
		httprouter.WithHealthCheckLivenessHandler(livenessHandler),
		httprouter.WithHealthCheckReadinessHandler(readinessHandler),
		httprouter.WithErrorHandlerFunc(errorHandlerFunc),
	)
}

func recordRequest(r *http.Request, status int, elapsed time.Duration, method, routePattern string, httpServerRequestDuration httpconv.ServerRequestDuration) {
	elapsedTimeSeconds := float64(elapsed) / float64(time.Second)

	attrs := []attribute.KeyValue{
		semconv.HTTPRequestMethodKey.String(method),
		semconv.HTTPRouteKey.String(routePattern),
		semconv.HTTPResponseStatusCode(status),
	}

	if status >= 500 {
		attrs = append(attrs, semconv.ErrorTypeKey.String("internal_error"))
	}

	httpServerRequestDuration.Record(r.Context(), elapsedTimeSeconds, httpconv.RequestMethodAttr(r.Method), r.Proto, attrs...)
}

var _patternReplacer = strings.NewReplacer(
	"{", "_",
	"}", "",
)

// SanitizeMetricTagValue sanitizes the given value in a standard way. It:
//   - Trims suffix "/".
//   - Replace "{" with "_"
//   - Remove  "}".
func sanitizeMetricTagValue(value string) string {
	value = strings.TrimRight(value, "/")
	return _patternReplacer.Replace(value)
}

// printBanner prints an ASCII art banner with application information at startup
func (a *Application) printBanner() {
	appVersion := os.Getenv("APP_VERSION")
	if appVersion == "" {
		appVersion = "0.0.0"
	}

	fmt.Println(`
 #####  #######       #    ######  ######  
#     # #     #      # #   #     # #     # 
#       #     #     #   #  #     # #     # 
#  #### #     #    #     # ######  ######  
#     # #     #    ####### #       #       
#     # #     #    #     # #       #       
 #####  #######    #     # #       #       

Application Name: ` + a.Name + `
Application Version: ` + appVersion + `
 :: Go Web Application Library ::
`)
}

// responseWriterWrapper wraps http.ResponseWriter to capture response body and status code
type responseWriterWrapper struct {
	http.ResponseWriter
	buffer     *bytes.Buffer
	statusCode int
}

func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	// Write to buffer if it exists (for response logging)
	if w.buffer != nil {
		w.buffer.Write(data)
	}
	return w.ResponseWriter.Write(data)
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
