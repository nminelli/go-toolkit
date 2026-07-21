package httprouter

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// URLParam retrieves the value of the specified parameter from the given HTTP request.
// It uses the chi.URLParam function to extract the parameter value.
func URLParam(r *http.Request, param string) string {
	return chi.URLParam(r, param)
}
