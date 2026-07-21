package httprouter

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// DefaultTimeouts exports sane timeouts for Run.
var DefaultTimeouts = Timeouts{
	ShutdownTimeout: 10 * time.Second,
}

// Timeouts struct define different timeouts that Run takes into consideration
// when running the web server.
type Timeouts struct {
	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body.
	//
	// Because ReadTimeout does not let Handlers make per-request
	// decisions on each request body's acceptable deadline or
	// upload rate, most users will prefer to use
	// ReadHeaderTimeout. It is valid to use them both.
	ReadTimeout time.Duration

	// ReadHeaderTimeout is the amount of time allowed to read
	// request headers. The connection's read deadline is reset
	// after reading the headers and the Handler can decide what
	// is considered too slow for the body. If ReadHeaderTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	ReadHeaderTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read. Like ReadTimeout, it does not
	// let Handlers make decisions on a per-request basis.
	WriteTimeout time.Duration

	// ShutdownTimeout is the maximum duration for the server
	// to gracefully shutdown.
	ShutdownTimeout time.Duration
}

// Run executes the given handler h on the given net.Listener ln with the given
// timeouts. It blocks until SIGTERM o SIGINT is received by the running process.
func Run(ln net.Listener, timeouts Timeouts, h http.Handler) error {
	// Create a new server and set timeout values.
	server := http.Server{
		ReadTimeout:       timeouts.ReadTimeout,
		ReadHeaderTimeout: timeouts.ReadHeaderTimeout,
		WriteTimeout:      timeouts.WriteTimeout,
		Handler:           h,
	}

	return run(&server, timeouts.ShutdownTimeout, ln, false)
}

// RunTLS executes the given handler h on the given net.Listener ln with the given
// timeouts and tlsConfig. It blocks until SIGTERM o SIGINT is received by the running process.
func RunTLS(ln net.Listener, timeouts Timeouts, h http.Handler, tlsConfig *tls.Config) error {
	// Create a new server and set timeout values and tlsConfig.
	server := http.Server{
		ReadTimeout:       timeouts.ReadTimeout,
		ReadHeaderTimeout: timeouts.ReadHeaderTimeout,
		WriteTimeout:      timeouts.WriteTimeout,
		Handler:           h,
		TLSConfig:         tlsConfig,
	}

	return run(&server, timeouts.ShutdownTimeout, ln, true)
}

// RunWithShutdownHooks executes the given handler h on the given net.Listener ln with the given
// timeouts and shutdown hooks. The shutdown hooks are executed before server shutdown.
// It blocks until SIGTERM o SIGINT is received by the running process.
func RunWithShutdownHooks(ln net.Listener, timeouts Timeouts, h http.Handler, shutdownHooks ...func()) error {
	// Create a new server and set timeout values.
	server := http.Server{
		ReadTimeout:       timeouts.ReadTimeout,
		ReadHeaderTimeout: timeouts.ReadHeaderTimeout,
		WriteTimeout:      timeouts.WriteTimeout,
		Handler:           h,
	}

	return runWithShutdownHooks(&server, timeouts.ShutdownTimeout, ln, false, shutdownHooks...)
}

// RunTLSWithShutdownHooks executes the given handler h on the given net.Listener ln with the given
// timeouts, tlsConfig and shutdown hooks. The shutdown hooks are executed before server shutdown.
// It blocks until SIGTERM o SIGINT is received by the running process.
func RunTLSWithShutdownHooks(ln net.Listener, timeouts Timeouts, h http.Handler, tlsConfig *tls.Config, shutdownHooks ...func()) error {
	// Create a new server and set timeout values and tlsConfig.
	server := http.Server{
		ReadTimeout:       timeouts.ReadTimeout,
		ReadHeaderTimeout: timeouts.ReadHeaderTimeout,
		WriteTimeout:      timeouts.WriteTimeout,
		Handler:           h,
		TLSConfig:         tlsConfig,
	}

	return runWithShutdownHooks(&server, timeouts.ShutdownTimeout, ln, true, shutdownHooks...)
}

func run(server *http.Server, shutdownTimeout time.Duration, ln net.Listener, serveTLS bool) error {
	return runWithShutdownHooks(server, shutdownTimeout, ln, serveTLS)
}

func runWithShutdownHooks(server *http.Server, shutdownTimeout time.Duration, ln net.Listener, serveTLS bool, shutdownHooks ...func()) error {
	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		if serveTLS {
			serverErrors <- server.ServeTLS(ln, "", "")
		} else {
			serverErrors <- server.Serve(ln)
		}
	}()

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return fmt.Errorf("error in serve: %w", err)
	case <-shutdown:
		// Execute shutdown hooks before shutting down the server
		for _, hook := range shutdownHooks {
			if hook != nil {
				hook()
			}
		}

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		// Asking listener to shut down and shed load.
		err := server.Shutdown(ctx)
		if err == nil {
			return nil
		}

		// If there was an error when shutting down the server (or it timed out)
		// then we have to force it to stop.
		if err := server.Close(); err != nil {
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}
