package httprouter_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MFN-AISystems/go-toolkit/httprouter"
)

func TestBind_JSON(t *testing.T) {
	type s struct {
		Field1 string `json:"field1" validate:"required"`
	}

	type s2 struct {
		s
		Field2 []string `json:"field2" validate:"required"`
	}

	type s3 []json.RawMessage

	tt := []struct {
		name               string
		input              string
		expectedErr        string
		expectedStatusCode int
		destination        interface{}
		assertFunc         func(d interface{})
	}{
		{
			name:               "should return bad request when body is nil",
			expectedErr:        "400 bad_request: Request body is empty",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "should return unmarshal type error when json is valid but dest is not",
			input:              `{"field1":"1", "field2":"2"}`,
			expectedErr:        "400 bad_request: Unmarshal type error: expected=[]string, got=string, field=field2, offset=27",
			destination:        &s2{},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "should return syntax error when json is not valid",
			input:              "invalid content",
			expectedErr:        "400 bad_request: Syntax error: offset=1, error=invalid character 'i' looking for beginning of value",
			destination:        &s{},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "should return json unmarshal error",
			input:              `{}`,
			expectedErr:        "400 bad_request: json: Unmarshal(non-pointer chan string)",
			destination:        make(chan string),
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:        "should bind s",
			input:       `{"field1":"1"}`,
			destination: &s{},
			assertFunc: func(d interface{}) {
				s, ok := d.(*s)
				require.True(t, ok)
				require.Equal(t, s.Field1, "1")
			},
		},
		{
			name:        "should bind s2",
			input:       `{"field1":"1", "field2": ["1","2"]}`,
			destination: &s2{},
			assertFunc: func(d interface{}) {
				s, ok := d.(*s2)
				require.True(t, ok)
				require.Equal(t, s.Field1, "1")
				require.Len(t, s.Field2, 2)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := &http.Request{
				Body: io.NopCloser(strings.NewReader(tc.input)),
			}

			err := httprouter.Bind(req, tc.destination)
			if tc.expectedErr != "" {
				require.EqualError(t, err, tc.expectedErr)
				webErr := err.(*httprouter.Error)
				require.Equal(t, tc.expectedStatusCode, webErr.StatusCode)
			} else {
				require.NoError(t, err)
				tc.assertFunc(tc.destination)
			}
		})
	}
}

func TestBind_JSON_Errors(t *testing.T) {
	type s struct {
		Field1 string `json:"field1" validate:"required"`
	}

	tt := []struct {
		name          string
		input         string
		destination   any
		runAssertions func(t *testing.T, err error)
	}{
		{
			name:        "should return validation error when missing field value",
			input:       `{"field1":""}`,
			destination: &s{},
			runAssertions: func(t *testing.T, err error) {
				var validationErrors validator.ValidationErrors
				assert.True(t, errors.As(err, &validationErrors))
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", bytes.NewBufferString(tc.input))
			err := httprouter.Bind(req, tc.destination)
			tc.runAssertions(t, err)
		})
	}
}

func TestBind_UnsupportedMediaType(t *testing.T) {
	h := http.Header{}
	h.Add("Content-Type", "application/xml")
	r := http.Request{
		Header: h,
	}

	err := httprouter.Bind(&r, nil)
	webErr, ok := err.(*httprouter.Error)
	require.True(t, ok)
	require.Equal(t, http.StatusUnsupportedMediaType, webErr.StatusCode)
}
