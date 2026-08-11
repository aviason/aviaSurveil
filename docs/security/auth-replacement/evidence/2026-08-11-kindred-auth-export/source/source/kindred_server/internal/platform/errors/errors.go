package errors

import (
	"errors"
	"net/http"
)

type AppError struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func Wrap(err error, status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Err: err}
}

func BadRequest(message string) *AppError { return New(http.StatusBadRequest, "bad_request", message) }
func Unauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, "unauthorized", message)
}
func Forbidden(message string) *AppError { return New(http.StatusForbidden, "forbidden", message) }
func NotFound(message string) *AppError  { return New(http.StatusNotFound, "not_found", message) }
func Conflict(message string) *AppError  { return New(http.StatusConflict, "conflict", message) }
func Internal(err error) *AppError {
	return Wrap(err, http.StatusInternalServerError, "internal_error", "internal server error")
}

func ToAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal(err)
}
