package errors

import "web-analyzer/internal/errors"

// MiddlewareError is the type of errors thrown by middleware.
type MiddlewareError struct {
	*errors.GenericError
}

// NewMiddlewareError creates a new MiddlewareError instance.
func NewMiddlewareError(code, msg string, errs error) error {
	return &MiddlewareError{
		GenericError: errors.NewGenericError("MiddlewareError", code, msg, errs),
	}
}
