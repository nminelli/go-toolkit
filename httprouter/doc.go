/*
Package httprouter provides lightweight but powerful primitives for building web applications in Go.
It's designed to be simple, flexible, and modular, allowing you to use its components independently
rather than forcing you into a specific framework architecture.

Features:
  - HTTP routing with parameter support built on chi router
  - Middleware pipeline for request processing
  - Request binding (JSON) with validation
  - JSON response helpers
  - Standardized error handling
  - Circuit breaker middleware
  - Header forwarding for distributed tracing
  - Server initialization with graceful shutdown
  - Shutdown hooks for resource cleanup during graceful shutdown

Basic Usage:

	router := httprouter.New()

	router.Get("/hello", func(w http.ResponseWriter, r *http.Request) error {
		return httprouter.RespondJSON(w, http.StatusOK, map[string]string{
			"message": "Hello, World!",
		})
	})

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	if err := httprouter.Run(ln, httprouter.DefaultTimeouts, router); err != nil {
		log.Fatal(err)
	}

Shutdown Hooks:

Register functions to be executed during graceful shutdown:

	// Example: SQS poller shutdown
	shutdownHook := func() {
		fmt.Println("Stopping SQS poller...")
		poller.Stop()
	}

	if err := httprouter.RunWithShutdownHooks(ln, httprouter.DefaultTimeouts, router, shutdownHook); err != nil {
		log.Fatal(err)
	}

The router supports middleware, route groups, parameter extraction, and automatic health check endpoints.

Author: Nicolás Minelli <nicolash@cobre.co>
*/
package httprouter
