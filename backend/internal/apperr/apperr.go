package apperr

import "net/http"

// Error is what handlers/services return. Only Code, Message and Fields
// ever reach the client. Cause is logged server-side and NEVER sent.
type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
	Cause   error
}

func (e *Error) Error() string { return e.Code }
func (e *Error) Unwrap() error { return e.Cause } // lets errors.Is/As see the cause

func (e *Error) WithCause(err error) *Error {
	e.Cause = err
	return e
}

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func BadRequest() *Error { return New(http.StatusBadRequest, "INVALID_REQUEST", "Invalid request.") }
func Unauthorized() *Error {
	return New(http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.")
}
func Forbidden() *Error { return New(http.StatusForbidden, "FORBIDDEN", "Access denied.") }
func NotFound() *Error  { return New(http.StatusNotFound, "NOT_FOUND", "Resource not found.") }

func Conflict(code, message string) *Error { return New(http.StatusConflict, code, message) }
func Unprocessable(code, message string) *Error {
	return New(http.StatusUnprocessableEntity, code, message)
}

func Validation(fields map[string]string) *Error {
	e := New(http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed.")
	e.Fields = fields
	return e
}

func Internal(cause error) *Error {
	return New(http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.").WithCause(cause)
}
