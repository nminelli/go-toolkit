package webapp

import (
	"fmt"
	"net/http"
)

type BusinessError interface {
	error
	Code() string
	Description() string
	Unwrap() error
}

type HttpError interface {
	BusinessError
	HttpStatus() int
}

type httpError struct {
	status      int
	code        string
	description string
	wrapped     error
}

func (e *httpError) Error() string {
	if e.wrapped != nil {
		return fmt.Sprintf("%s: %s: %v", e.code, e.description, e.wrapped)
	}
	return fmt.Sprintf("%s: %s", e.code, e.description)
}

func (e *httpError) HttpStatus() int {
	return e.status
}

func (e *httpError) Code() string {
	return e.code
}

func (e *httpError) Description() string {
	return e.description
}

func (e *httpError) Unwrap() error {
	return e.wrapped
}

func NewHttpError(status int, code, description string, err error) HttpError {
	return &httpError{
		status:      status,
		code:        code,
		description: description,
		wrapped:     err,
	}
}

func NewBadRequestError(code, description string, err error) HttpError {
	return NewHttpError(http.StatusBadRequest, code, description, err)
}

func NewConflictError(code, description string, err error) HttpError {
	return NewHttpError(http.StatusConflict, code, description, err)
}

func NewNotFoundError(code, description string) HttpError {
	return NewHttpError(http.StatusNotFound, code, description, nil)
}

func NewInternalServerError(code, description string, err error) HttpError {
	return NewHttpError(http.StatusInternalServerError, code, description, err)
}
