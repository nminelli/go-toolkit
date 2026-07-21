/*
Package webapp provides a robust framework for building web applications in Go. It offers a simple
and efficient way to bootstrap HTTP servers with sensible defaults, telemetry, structured logging,
environment awareness, and other essential features for production-ready web services.

Features:
  - Simple web application setup with minimal configuration
  - Telemetry integration with logging, metrics, and tracing
  - Pre-configured middleware stack for common concerns
  - Health check endpoints (liveness and readiness)
  - Customizable router based on the chi router
  - Consistent error response formatting
  - Request compression based on content type
  - Configurable HTTP server timeouts
  - Shutdown hooks for graceful resource cleanup

Basic Usage:

	app, err := webapp.New()
	if err != nil {
		panic(err)
	}

	app.Router.Get("/hello", func(w http.ResponseWriter, r *http.Request) error {
		return httprouter.RespondJSON(w, http.StatusOK, map[string]string{"message": "Hello, World!"})
	})

	if err := app.Run(); err != nil {
		panic(err)
	}

Shutdown Hooks:

Register shutdown hooks for graceful resource cleanup:

	// Example: SQS poller integration
	app, err := webapp.New(
		webapp.WithShutdownHooks(poller.Stop),
	)

	// Add individual shutdown hooks
	app2, err := webapp.New(
		webapp.WithShutdownHook(func() {
			fmt.Println("Cleanup task 1")
		}),
		webapp.WithShutdownHook(func() {
			fmt.Println("Cleanup task 2")
		}),
	)

	// Or add hooks after creation
	app.AddShutdownHook(func() {
		// Custom cleanup logic
	})

The webapp package includes built-in middleware for telemetry, logging, panic recovery, header forwarding,
and compression. Health check endpoints are automatically available at /actuator/health/liveness and
/actuator/health/readiness.

Author: Nicolás Minelli <nicolash@cobre.co>
*/
package webapp
