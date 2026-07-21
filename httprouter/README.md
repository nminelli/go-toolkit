# HTTP Router Package

![technology Go](https://img.shields.io/badge/go-1.24+-blue.svg)

The `httprouter` package provides lightweight but powerful primitives for building web applications in Go.
It's designed to be simple, flexible, and modular,
allowing you to use its components independently
rather than forcing you into a specific framework architecture.

## Table of Contents

- [Installation](#installation)
- [Core Features](#core-features)
- [Basic Usage](#basic-usage)
- [Router](#router)
  - [Creating a Router](#creating-a-router)
  - [Route Configuration](#route-configuration)
  - [Route Parameters](#route-parameters)
  - [Middleware Support](#middleware-support)
  - [Router Groups](#router-groups)
  - [Subrouters](#subrouters)
  - [Health Checks and Profiling](#health-checks-and-profiling)
- [Request Handling](#request-handling)
  - [Handlers and Error Handling](#handlers-and-error-handling)
  - [URL Parameters](#url-parameters)
  - [Binding Request Data](#binding-request-data)
  - [Responding with JSON](#responding-with-json)
- [Error Handling](#error-handling)
  - [Creating Custom Errors](#creating-custom-errors)
  - [Error Handler Functions](#error-handler-functions)
- [Middleware](#middleware)
  - [Circuit Breaker](#circuit-breaker)
  - [Client ID Extraction](#client-id-extraction)
  - [Audited Middleware](#audited-middleware)
  - [Header Forwarding](#header-forwarding)
  - [Using Custom Middleware](#using-custom-middleware)
- [Running Your Server](#running-your-server)
  - [Basic Server](#basic-server)
- [Running Your Server](#running-your-server)
  - [Basic Server](#basic-server)
  - [TLS Server](#tls-server)
  - [Timeouts Configuration](#timeouts-configuration)
  - [Graceful Shutdown](#graceful-shutdown)

## Installation

```bash
go get github.com/nminelli/go-toolkit/httprouter
```

## Core Features

- HTTP routing with parameter support
- Middleware pipeline for request processing
- Request binding (JSON)
- JSON response helpers
- Standardized error handling
- Circuit breaker middleware
- Client ID extraction middleware (from headers or JWT tokens)
- Audit logging middleware (captures and publishes request/response data to Kafka)
- Header forwarding for distributed tracing
- Server initialization with graceful shutdown

## Basic Usage

Here's a minimal example to get started:

```go
package main

import (
	"log"
	"net"
	"net/http"

	"github.com/nminelli/go-toolkit/httprouter"
)

func main() {
	// Create a new router
	router := httprouter.New()

	// Register a route
	router.Get("/hello", func(w http.ResponseWriter, r *http.Request) error {
		return httprouter.RespondJSON(w, http.StatusOK, map[string]string{
			"message": "Hello, World!",
		})
	})

	// Start the server
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Server started on :8080")
	if err := httprouter.Run(ln, httprouter.DefaultTimeouts, router); err != nil {
		log.Fatal(err)
	}
}
```

## Router

The router is built on top of [chi](https://github.com/go-chi/chi), providing a familiar and flexible routing system.

### Creating a Router

```go
// Create a basic router
router := httprouter.New()

// Create a router with additional options
router := httprouter.New(
    httprouter.WithErrorHandlerFunc(customErrorHandler),
    httprouter.WithNotFoundHandler(customNotFoundHandler),
    httprouter.WithHealthCheckLivenessHandler(livenessHandler),
    httprouter.WithHealthCheckReadinessHandler(readinessHandler),
    httprouter.WithEnableProfiler(true),
    httprouter.WithGlobalMiddlewares(middleware1, middleware2),
)
```

### Route Configuration

Register routes for different HTTP methods:

```go
router.Get("/users", listUsersHandler)
router.Post("/users", createUserHandler)
router.Put("/users/{id}", updateUserHandler)
router.Patch("/users/{id}", patchUserHandler)
router.Delete("/users/{id}", deleteUserHandler)
router.Options("/users", optionsHandler)
router.Head("/users", headHandler)
router.Connect("/connect", connectHandler)
router.Trace("/trace", traceHandler)
```

### Route Parameters

Extract URL parameters from requests:

```go
router.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) error {
    id := httprouter.URLParam(r, "id")
    // Use id...
    return httprouter.RespondJSON(w, http.StatusOK, map[string]string{"id": id})
})
```

### Middleware Support

Apply middleware to the entire router:

```go
router.Use(middleware.Logger)
router.Use(middleware.Recoverer)
```

### Router Groups

Group routes with shared middleware:

```go
router.Group(func(r httprouter.Router) {
    // Add middleware specific to this group
    r.Use(authMiddleware)

    // Define routes for this group
    r.Get("/admin/dashboard", dashboardHandler)
    r.Get("/admin/users", adminUsersHandler)
})
```

### Subrouters

Mount subrouters for specific URL paths:

```go
router.Route("/api/v1", func(r httprouter.Router) {
    r.Get("/users", listUsersHandler)
    r.Post("/users", createUserHandler)

    // Nested routes
    r.Route("/users/{id}", func(r httprouter.Router) {
        r.Get("/", getUserHandler)
        r.Put("/", updateUserHandler)
        r.Delete("/", deleteUserHandler)
    })
})
```

### Health Checks and Profiling

The router automatically sets up health check endpoints and profiling when configured:

```go
router := httprouter.New(
    httprouter.WithHealthCheckLivenessHandler(livenessHandler),
    httprouter.WithHealthCheckReadinessHandler(readinessHandler),
    httprouter.WithEnableProfiler(true),
)
```

This will create:
- `/actuator/health/liveness` endpoint for liveness probes
- `/actuator/health/readiness` endpoint for readiness probes
- `/actuator/health` as an alias for readiness
- `/debug` endpoint for profiling with pprof

## Request Handling

### Handlers and Error Handling

The package defines a custom `Handler` type for consistent error handling:

```go
// Handler function signature
type Handler func(w http.ResponseWriter, r *http.Request) error

// Example implementation
func getUserHandler(w http.ResponseWriter, r *http.Request) error {
    id := httprouter.URLParam(r, "id")
    
    user, err := userService.FindByID(id)
    if err != nil {
        return httprouter.NewErrorf(http.StatusNotFound, "User not found: %s", id)
    }
    
    return httprouter.RespondJSON(w, http.StatusOK, user)
}
```

### URL Parameters

Extract URL parameters from requests:

```go
func getUserHandler(w http.ResponseWriter, r *http.Request) error {
    id := httprouter.URLParam(r, "id")
    // Process using id...
    return nil
}
```

### Binding Request Data

This library makes use of [validator](https://github.com/go-playground/validator) library for request validation. This means you can use struct tags to define validation rules and that the same validation errors are returned.

Bind JSON request body to Go structs:

```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=0,lt=130"`
}

func createUserHandler(w http.ResponseWriter, r *http.Request) error {
    var req CreateUserRequest
    if err := httprouter.Bind(r, &req); err != nil {
        return err // Automatically handles validation errors
    }
    
    // Process the validated request...
    user, err := userService.Create(req)
    if err != nil {
        return err
    }
    
    return httprouter.RespondJSON(w, http.StatusCreated, user)
}
```

### Responding with JSON

Send JSON responses:

```go
func getDataHandler(w http.ResponseWriter, r *http.Request) error {
    data := map[string]interface{}{
        "id": 1,
        "name": "Example",
        "active": true,
    }
    
    return httprouter.RespondJSON(w, http.StatusOK, data)
}

// Returning no content
func deleteHandler(w http.ResponseWriter, r *http.Request) error {
    // Delete operation...
    return httprouter.RespondJSON(w, http.StatusNoContent, nil)
}
```

## Error Handling

### Creating Custom Errors

Create errors with appropriate HTTP status codes:

```go
// Simple error
err := httprouter.NewError(http.StatusNotFound, "User not found")

// Formatted error
err := httprouter.NewErrorf(http.StatusBadRequest, "Invalid user ID: %s", id)
```

### Error Handler Functions

Define custom error handling logic:

```go
func customErrorHandler(err error, defaultHandler func(error) httprouter.HandlerError) httprouter.HandlerError {
    // Process specific error types
    if errors.Is(err, sql.ErrNoRows) {
        return httprouter.HandlerError{
            StatusCode: http.StatusNotFound,
            Error: map[string]string{
                "code": "not_found",
                "message": "Resource not found",
            },
            Notify: false, // Don't notify monitoring systems
        }
    }

    // Fall back to default handler for other errors
    return defaultHandler(err)
}

// Use it when creating the router
router := httprouter.New(httprouter.WithErrorHandlerFunc(customErrorHandler))
```

## Middleware

### Circuit Breaker

Add a circuit breaker to protect your service:

```go
// Create a circuit breaker (implement the CircuitBreaker interface)
cb := &MyCircuitBreaker{}

// Add the circuit breaker middleware
router.Use(httprouter.Breaker(cb, httprouter.DefaultBreakerValidator))
```

Example implementation:

```go
type MyCircuitBreaker struct {
    failures int
    threshold int
    open bool
}

func (cb *MyCircuitBreaker) Allow() bool {
    return !cb.open
}

func (cb *MyCircuitBreaker) Success() {
    cb.failures = 0
}

func (cb *MyCircuitBreaker) Failure() {
    cb.failures++
    if cb.failures >= cb.threshold {
        cb.open = true
    }
}
```

### Client ID Extraction

Extract client IDs from headers or JWT tokens for request processing:

```go
import (
    "github.com/nminelli/go-toolkit/httprouter/middleware"
)

// Extract client ID from header
router.Use(middleware.ClientID(middleware.SourceHeader))

// Extract client ID from JWT token
router.Use(middleware.ClientID(middleware.SourceJWT))
```

The middleware supports two sources for client ID extraction:

1. **Header-based extraction** (`SourceHeader`): Extracts the client ID from the `client_id` header
2. **JWT-based extraction** (`SourceJWT`): Extracts the client ID from the JWT token in the `Authorization` header

For JWT extraction, the middleware:
- Validates the JWT token structure (without signature verification)
- Ensures the `custom:type` claim equals "api_user"
- Extracts the client ID from the `custom:linked_clients` claim
- Handles pipe-separated values (uses the first part before the pipe)

Once extracted, the client ID is stored in the request context and can be retrieved in handlers:

```go
func getUserHandler(w http.ResponseWriter, r *http.Request) error {
    clientID := middleware.GetClientIDFromContext(r.Context())
    if clientID == "" {
        return httprouter.NewError(http.StatusUnauthorized, "Client ID not found")
    }
    
    // Use clientID for business logic...
    return httprouter.RespondJSON(w, http.StatusOK, map[string]string{
        "client_id": clientID,
        "message": "Authenticated request",
    })
}
```

If the client ID cannot be extracted or validated, the middleware returns a 401 Unauthorized response.

### Audited Middleware

The Audited middleware captures and logs audit events for each HTTP request and response. It records request/response details, including headers and bodies, and sends them to a Kafka topic for auditing purposes.

```go
import (
    "github.com/nminelli/go-toolkit/httprouter/middleware"
)

// Add audited middleware to track all HTTP requests
router.Use(middleware.Audited())
```

**Required Environment Variables:**

- `AUDITED_KAFKA_CREDENTIALS_SECRET_NAME`: Name of the AWS Secrets Manager secret containing Kafka credentials
- `AUDITED_KAFKA_BOOTSTRAP_SERVER`: Kafka bootstrap server URL
- `AUDITED_KAFKA_SCHEMA_REGISTRY_URL`: URL of the Kafka Schema Registry
- `COMPONENT_NAME`: Name of your service/component (used in audit events)
- `ENVIRONMENT`: Deployment environment (e.g., `dev`, `prod`, `test`)

### Header Forwarding

Forward tracing headers for distributed systems:

```go
import (
    "github.com/nminelli/go-toolkit/httprouter/middleware"
)

// Add header forwarding middleware
router.Use(middleware.HeaderForwarder)
```

### Using Custom Middleware

Implement your own middleware:

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Call the next handler
        next.ServeHTTP(w, r)
        
        // Log after request is processed
        log.Printf(
            "%s %s completed in %v",
            r.Method,
            r.URL.Path,
            time.Since(start),
        )
    })
}

// Apply to the router
router.Use(loggingMiddleware)
```

## Running Your Server

### Basic Server

```go
// Create a listener
ln, err := net.Listen("tcp", ":8080")
if err != nil {
    log.Fatal(err)
}

// Run the server with default timeouts
if err := httprouter.Run(ln, httprouter.DefaultTimeouts, router); err != nil {
    log.Fatal(err)
}
```

### TLS Server

```go
// Create a listener
ln, err := net.Listen("tcp", ":443")
if err != nil {
    log.Fatal(err)
}

// Configure TLS
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS12,
    // Add other TLS config options as needed
}

// Run the TLS server
if err := httprouter.RunTLS(ln, httprouter.DefaultTimeouts, router, tlsConfig); err != nil {
    log.Fatal(err)
}
```

### Timeouts Configuration

Configure custom timeouts:

```go
timeouts := httprouter.Timeouts{
    ReadTimeout:       5 * time.Second,
    ReadHeaderTimeout: 2 * time.Second,
    WriteTimeout:      10 * time.Second,
    ShutdownTimeout:   15 * time.Second,
}

if err := httprouter.Run(ln, timeouts, router); err != nil {
    log.Fatal(err)
}
```

### Graceful Shutdown

The `Run` and `RunTLS` functions handle graceful shutdown automatically when they receive a termination signal (SIGTERM or SIGINT).

```go
// This will automatically:
// 1. Listen for termination signals
// 2. Stop accepting new requests
// 3. Allow existing requests to complete (up to ShutdownTimeout)
// 4. Close the server cleanly
if err := httprouter.Run(ln, httprouter.DefaultTimeouts, router); err != nil {
    log.Fatal(err)
}
```

### Shutdown Hooks

You can register shutdown hooks that will be executed before the HTTP server shutdown. This is useful for cleaning up resources like database connections, stopping background workers, or flushing data.

```go
// Example: SQS poller shutdown
shutdownHook := func() {
    fmt.Println("Stopping SQS poller...")
    poller.Stop()
    fmt.Println("SQS poller stopped")
}

// Run server with shutdown hooks
if err := httprouter.RunWithShutdownHooks(ln, httprouter.DefaultTimeouts, router, shutdownHook); err != nil {
    log.Fatal(err)
}
```

```go
// Multiple shutdown hooks
cleanupDB := func() {
    fmt.Println("Closing database connections...")
    db.Close()
}

stopWorkers := func() {
    fmt.Println("Stopping background workers...")
    workerPool.Stop()
}

// All hooks will be executed in order during shutdown
if err := httprouter.RunWithShutdownHooks(ln, httprouter.DefaultTimeouts, router, cleanupDB, stopWorkers); err != nil {
    log.Fatal(err)
}
```

TLS version with shutdown hooks:

```go
if err := httprouter.RunTLSWithShutdownHooks(ln, httprouter.DefaultTimeouts, router, tlsConfig, shutdownHook); err != nil {
    log.Fatal(err)
}
```

# Contribution

**Authors:**

  - Nicolas Minelli <nicolash@cobre.co>