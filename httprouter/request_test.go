package httprouter_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MFN-AISystems/go-toolkit/httprouter"
)

func TestURLParam(t *testing.T) {
	r := httprouter.New()

	// Set up a route that captures a named URL parameter
	r.Get("/{param}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		param := httprouter.URLParam(r, "param")
		_, _ = w.Write([]byte(param))
	}))

	testCases := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Simple",
			url:      "/test",
			expected: "test",
		},
		{
			name:     "Number",
			url:      "/123",
			expected: "123",
		},
		{
			name:     "SpecialCharacter",
			url:      "/test+test",
			expected: "test+test",
		},
		{
			name:     "Empty",
			url:      "/",
			expected: "404 page not found\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", tc.url, nil)
			responseRecorder := httptest.NewRecorder()

			r.ServeHTTP(responseRecorder, request)

			result := responseRecorder.Body.String()

			if result != tc.expected {
				t.Errorf("expected URLParam %q, got %q", tc.expected, result)
			}
		})
	}
}
