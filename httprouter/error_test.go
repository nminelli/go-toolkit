package httprouter_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nminelli/go-toolkit/httprouter"
)

func TestNewError(t *testing.T) {
	err := httprouter.NewError(http.StatusBadRequest, "error occurred")
	require.Error(t, err)
	require.EqualValues(t, "400 bad_request: error occurred", err.Error())
}

func TestNewErrorf(t *testing.T) {
	err := httprouter.NewErrorf(http.StatusBadRequest, "error occurred: %s", "detail")
	require.Error(t, err)
	require.EqualValues(t, "400 bad_request: error occurred: detail", err.Error())
}
