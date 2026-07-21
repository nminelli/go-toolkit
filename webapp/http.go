package webapp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/nminelli/go-toolkit/telemetry/log"
	"github.com/nminelli/go-toolkit/telemetry/tracing"
)

// httpErrorResponse represents the JSON structure returned to clients on errors
type httpErrorResponse struct {
	Code        string `json:"error_code"`
	Description string `json:"error_description"`
}

// JSON converts a Go value to JSON and sends it to the client.
// If value is nil or code is equal to http.StatusNoContent we avoid writing any content to w.
// HTTP response header with the provided status code is always set.
func JSON(_ context.Context, w http.ResponseWriter, status int, data interface{}) {
	respondJSON(w, status, data)
}

// JSONError sends a JSON error response to the client compliant with Cobre's API error format and
// guaranteeing that server errors (5xx) are recorded in the tracing system.
func JSONError(ctx context.Context, w http.ResponseWriter, err HttpError) {
	if err.HttpStatus() >= http.StatusInternalServerError {
		var recordableErr error = err
		if wrapped := err.Unwrap(); wrapped != nil {
			recordableErr = wrapped
		}
		tracing.RecordError(ctx, recordableErr)
	}

	respondJSON(w, err.HttpStatus(), httpErrorResponse{Code: err.Code(), Description: err.Description()})
}

// RespondJSON converts a Go value to JSON and sends it to the client.
// If value is nil or code is equal to http.StatusNoContent we avoid writing any content to w.
// HTTP response header with the provided status code is always set.
func respondJSON(w http.ResponseWriter, code int, value any) {
	// According to https://tools.ietf.org/search/rfc2616#section-7.2.1:
	//
	// "Any HTTP/1.1 message containing an entity-value SHOULD include a Content-Type
	// header field defining the media type of that value"
	//
	// Since there is no content, then there is no reason to specify a Content-Type header
	if code == http.StatusNoContent || value == nil {
		w.WriteHeader(code)
		return
	}

	var jsonData []byte

	var err error
	switch v := value.(type) {
	case []byte:
		jsonData = v
	case io.Reader:
		jsonData, err = io.ReadAll(v)
		if err != nil {
			log.Error(context.Background(), "failed to read json response", log.Err(err))
		}
	default:
		jsonData, err = json.Marshal(v)
		if err != nil {
			log.Error(context.Background(), "failed to marshal json response", log.Err(err))
		}
	}

	if err != nil {
		write(w, http.StatusInternalServerError, []byte(`{"message":"Internal Server Error"}`))
		return
	}

	// Send the result back to the client.
	write(w, code, jsonData)
}

func write(w http.ResponseWriter, status int, body []byte) {
	// Set the content type.
	w.Header().Set("Content-Type", "application/json")

	// Write the status code to the response and context.
	w.WriteHeader(status)

	_, err := w.Write(body)
	if err != nil {
		panic(err)
	}
}
