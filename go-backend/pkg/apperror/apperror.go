// Package apperror defines the application's custom error type.
//
// In Node.js (src/utils/AppError.js):
//   class AppError extends Error {
//     constructor(message, statusCode = 500) {
//       super(message);
//       this.statusCode = statusCode;
//     }
//   }
//
// In Go, we don't extend classes — we implement the built-in `error`
// interface instead. Any type that has an `Error() string` method
// satisfies the `error` interface automatically. This is Go's approach
// to polymorphism: interfaces, not inheritance.
package apperror

import "fmt"

// AppError represents a business-logic error that carries an HTTP status code.
// Unlike panic/recover (Go's emergency exception mechanism), AppError is a
// normal value returned through the call chain, never thrown.
//
// Concept: Interfaces in Go
//   The built-in `error` interface looks like this (from the standard library):
//     type error interface {
//       Error() string
//     }
//   Any type with an Error() string method IS an error. No declaration needed.
//   This is called "structural typing" or "duck typing" — if it walks like
//   an error and quacks like an error, it IS an error.
type AppError struct {
	Message    string // Human-readable error message
	StatusCode int    // HTTP status code (400, 401, 403, 404, 409, 500, 503...)
	Err        error  // Optional: the underlying cause (wrapping pattern)
}

// Error() makes AppError satisfy the built-in `error` interface.
// This method is called automatically whenever Go needs to display the error
// (e.g., in fmt.Println(err), log.Println(err), err.Error()).
func (e *AppError) Error() string {
	if e.Err != nil {
		// %w wraps the underlying error — callers can use errors.Is() / errors.As()
		// to inspect it without unpacking the whole chain manually.
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the wrapped error, enabling errors.Is() / errors.As()
// to work across the whole error chain. This is Go's standard error
// wrapping protocol (introduced in Go 1.13).
func (e *AppError) Unwrap() error {
	return e.Err
}

// --- Constructors ---
// These functions are like constructors in JS but as plain functions.
// By convention, Go constructors are named New<Type>.

// New creates a generic AppError.
func New(statusCode int, message string) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// Wrap creates an AppError that wraps an underlying error.
// Use this when you catch a low-level error and want to add context:
//   if err := db.Query(...); err != nil {
//       return apperror.Wrap(500, "querying users", err)
//   }
func Wrap(statusCode int, message string, err error) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}

// --- Shorthand constructors for common HTTP errors ---
// These mirror the most common Express patterns:
//   res.status(400).json({...})  →  apperror.BadRequest("message")
//   res.status(401).json({...})  →  apperror.Unauthorized("message")
//   etc.

func BadRequest(message string) *AppError      { return New(400, message) }
func Unauthorized(message string) *AppError    { return New(401, message) }
func Forbidden(message string) *AppError       { return New(403, message) }
func NotFound(message string) *AppError        { return New(404, message) }
func Conflict(message string) *AppError        { return New(409, message) }
func Internal(message string) *AppError        { return New(500, message) }
func ServiceUnavailable(message string) *AppError { return New(503, message) }

