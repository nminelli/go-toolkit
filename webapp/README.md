# Webapp Package

![technology Go](https://img.shields.io/badge/go-1.24+-blue.svg)

The `webapp` package provides a robust framework for building web applications in Go. It offers a simple and efficient way to bootstrap HTTP servers with sensible defaults, telemetry, structured logging, environment awareness, and other essential features for production-ready web services.

## Features

- **Simple Web Application Setup**: Create and run web applications with minimal configuration
- **Telemetry Integration**: Built-in support for logging, metrics, and tracing
- **Middleware Stack**: Pre-configured middleware for common concerns (logging, panic recovery, etc.)
- **Health Check Endpoints**: Ready-to-use liveness and readiness endpoints
- **Customizable Router**: Based on the powerful `chi` router with full flexibility
- **Error Handling**: Consistent error response formatting and error handling middleware
- **Request Compression**: Automatic response compression based on content type
- **Timeouts Configuration**: Control HTTP server timeouts easily

## Installation

```bash
go get github.com/MFN-AISystems/go-toolkit/webapp
```

## Basic Usage

### Creating and Running a Web Application

```go
package main

import (
	"context"
	"net/http"

	"github.com/MFN-AISystems/go-toolkit/httprouter"
	"github.com/MFN-AISystems/go-toolkit/webapp"
)

func main() {
	// Create a new web application with default settings
	app, err := webapp.New()
	if err != nil {
		panic(err)
	}

	// Add a route
	app.Router.Get("/hello", func(w http.ResponseWriter, r *http.Request) error {
		return httprouter.RespondJSON(w, http.StatusOK, map[string]string{"message": "Hello, World!"})
	})

	// Start the application
	if err := app.Run(); err != nil {
		panic(err)
	}
}
```

This example creates a web server that:
- Listens on port 8080 by default (configurable via the `PORT` environment variable)
- Handles requests to `/actuator/health/liveness` and `/actuator/health/readiness` for health checks

## Configuration Options

The `webapp` package offers several configuration options when creating a new application:

### Server Timeouts

```go
app, err := webapp.New(
    webapp.WithTimeouts(httprouter.Timeouts{
        ReadTimeout:       5 * time.Second,
        ReadHeaderTimeout: 5 * time.Second,
        WriteTimeout:      10 * time.Second,
        ShutdownTimeout:   10 * time.Second,
    }),
)
```

### Custom Network Listener

```go
listener, err := net.Listen("tcp", ":9090")
if err != nil {
    panic(err)
}

app, err := webapp.New(
    webapp.WithListener(listener),
)
```

### Custom Error Handler

```go
errHandler := func(err error, defaultHandlerError func(error) httprouter.HandlerError) httprouter.HandlerError {
    // Custom error handling logic
    return httprouter.HandlerError{
        Error:      err,
        Notify:     true,
        StatusCode: http.StatusInternalServerError,
    }
}

app, err := webapp.New(
    webapp.WithErrorHandler(errHandler),
)
```

### Custom Middlewares

```go
myMiddleware := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Custom logic before request
        next.ServeHTTP(w, r)
        // Custom logic after request
    })
}

app, err := webapp.New(
    webapp.WithGlobalMiddlewares(myMiddleware),
)
```

### Shutdown Hooks

Shutdown hooks allow you to register functions that will be executed during graceful shutdown, before the HTTP server is closed. This is particularly useful for cleaning up resources like stopping SQS pollers, closing database connections, or flushing buffers.

```go
import (
    "github.com/MFN-AISystems/go-toolkit/service/aws/sqs"
    "github.com/MFN-AISystems/go-toolkit/webapp"
)

// Create SQS poller
pollerConfig := sqs.PollerConfig{
    QueueURL:           "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue",
    DefaultHandlerName: "message-processor",
    MaxMsg:             10,
    WaitTimeSeconds:    20,
    MaxWorkers:         5,
}

processFn := func(ctx context.Context, msg types.Message) error {
    // Process the message
    return nil
}

poller, err := sqs.NewPoller(pollerConfig, processFn)
if err != nil {
    panic(err)
}

// Start the poller
poller.Start()

// Create webapp with shutdown hooks
app, err := webapp.New(
    // Register poller.Stop as a shutdown hook
    webapp.WithShutdownHooks(poller.Stop),
)
if err != nil {
    panic(err)
}

// You can also add individual shutdown hooks using WithShutdownHook
app2, err := webapp.New(
    webapp.WithShutdownHook(func() {
        fmt.Println("First cleanup task...")
    }),
    webapp.WithShutdownHook(func() {
        fmt.Println("Second cleanup task...")
    }),
)

// Or add shutdown hooks after creating the app
app.AddShutdownHook(func() {
    // Additional cleanup logic
    fmt.Println("Performing additional cleanup...")
})

// Start the application - when terminated, it will:
// 1. Stop the SQS poller gracefully
// 2. Execute any additional shutdown hooks
// 3. Shutdown the HTTP server
if err := app.Run(); err != nil {
    panic(err)
}
```

## Built-in Middleware

The `webapp` package comes with several built-in middleware components:

1. **Telemetry Middleware**: Initiates tracing spans for incoming requests
2. **Log Middleware**: Logs incoming requests with method, path, and remote address
3. **Panics Middleware**: Recovers from panics and returns appropriate error responses
4. **Header Forwarding Middleware**: Forwards specific headers for outgoing requests
5. **Compression Middleware**: Compresses response body based on Accept-Encoding header

## Health Check Endpoints

By default, the `webapp` package sets up health check endpoints:

- `/actuator/health/liveness`: For liveness probes
- `/actuator/health/readiness`: For readiness probes
- `/actuator/health`: For readiness probes

These endpoints return 204 No Content when the application is healthy.

## Telemetry

The `webapp` package integrates with the telemetry package for:

1. **Logging**: Structured logging with configurable log levels
2. **Metrics**: HTTP request metrics (count, duration) with useful attributes
3. **Tracing**: Distributed tracing support for request flows

## Example: Complete Web Application

```go
package main

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/MFN-AISystems/go-toolkit/httprouter"
	"github.com/MFN-AISystems/go-toolkit/webapp"
)

func main() {
	// Create a custom listener
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(err)
	}

	// Custom middleware
	requestID := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := uuid.New().String()
			w.Header().Set("X-Request-ID", id)
			next.ServeHTTP(w, r)
		})
	}

	// Create the application with custom settings
	app, err := webapp.New(
		webapp.WithListener(listener),
		webapp.WithTimeouts(httprouter.Timeouts{
			ReadTimeout:       10 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      10 * time.Second,
			ShutdownTimeout:   15 * time.Second,
		}),
		webapp.WithGlobalMiddlewares(requestID),
	)
	if err != nil {
		panic(err)
	}

	// Add routes
	app.Router.Get("/api/users", getUsers)
	app.Router.Post("/api/users", createUser)

	// Start the application
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func getUsers(w http.ResponseWriter, r *http.Request) error {
	users := []map[string]interface{}{
		{"id": 1, "name": "User 1"},
		{"id": 2, "name": "User 2"},
	}
	return httprouter.RespondJSON(w, http.StatusOK, users)
}

func createUser(w http.ResponseWriter, r *http.Request) error {
	var user map[string]interface{}
	if err := httprouter.BindJSON(r, &user); err != nil {
		return err
	}

	// Process the user...

	return httprouter.RespondJSON(w, http.StatusCreated, user)
}
```

## Accessing Components

The `Application` struct provides access to its components:

```go
// Access the router to add routes
app.Router.Get("/path", handler)

// Access the logger
app.Logger.Info("message", "key", "value")
```

## Advanced Usage: Customizing Routes and Handlers

```go
// Group routes
app.Router.Route("/api/v1", func(r chi.Router) {
    r.Get("/resources", listResources)
    r.Post("/resources", createResource)
    r.Route("/resources/{id}", func(r chi.Router) {
        r.Get("/", getResource)
        r.Put("/", updateResource)
        r.Delete("/", deleteResource)
    })
})

// Using URL parameters
func getResource(w http.ResponseWriter, r *http.Request) error {
    id := chi.URLParam(r, "id")
    // ... get resource by id
    return httprouter.RespondJSON(w, http.StatusOK, resource)
}
```

## Error Handling

The `webapp` package provides a consistent way to handle errors:

```go
func handler(w http.ResponseWriter, r *http.Request) error {
    // Return an HTTP error
    return httprouter.NewErrorf(http.StatusNotFound, "Resource not found")
    
    // Or return any error, it will be handled appropriately
    return errors.New("something went wrong")
}
```

By default, the error handler:
- Converts `httprouter.Error` to the appropriate HTTP status code
- Treats other errors as 500 Internal Server Error
- Returns error responses in a consistent JSON format

# Contribution

**Authors:**

- Nicolas Minelli <minellinh@gmail.com>
