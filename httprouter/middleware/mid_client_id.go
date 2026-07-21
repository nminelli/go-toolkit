package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MFN-AISystems/go-toolkit/telemetry/log"
)

type middlewareError struct {
	Message string `json:"message"`
}

const (
	SourceHeader = "Header"
	SourceJWT    = "JWT"
)

type clientIDKey struct{}

func ClientID(source string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var clientID string
			var err error

			switch source {
			case SourceHeader:
				clientID, err = retrieveClientIDFromHeader(r)
			default:
				clientID, err = retrieveClientIDFromJWT(r)
			}

			if err != nil {
				log.Error(r.Context(), fmt.Sprintf("Failed to retrieve client ID: %v", err), log.Err(err))
				respondJSON(w, http.StatusUnauthorized, middlewareError{Message: "Unauthorized"})
				return
			}

			// Add the client ID to the request context
			ctx := AddClientIDToContext(r.Context(), clientID)
			r = r.WithContext(ctx)

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// GetClientIDFromContext retrieves the client ID from the request context
func GetClientIDFromContext(ctx context.Context) string {
	clientID, ok := ctx.Value(clientIDKey{}).(string)
	if ok {
		return clientID
	}

	log.Info(ctx, "No client ID found in context")
	return ""
}

// AddClientIDToContext adds the client ID to the context
func AddClientIDToContext(ctx context.Context, clientID string) context.Context {
	return context.WithValue(ctx, clientIDKey{}, clientID)
}

func retrieveClientIDFromHeader(r *http.Request) (string, error) {
	clientID := r.Header.Get("client_id")
	if clientID == "" {
		return "", fmt.Errorf("client_id header missing")
	}

	return clientID, nil
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Error(context.Background(), "Failed to marshal JSON response", log.Err(err))
		write(w, http.StatusInternalServerError, []byte(`{"message":"Internal Server Error"}`))
		return
	}

	// Set the content type.
	w.Header().Set("Content-Type", "application/json")

	// Write the status code to the response and context.
	write(w, status, body)
}

func write(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err := w.Write(body)
	if err != nil {
		panic(err)
	}
}
