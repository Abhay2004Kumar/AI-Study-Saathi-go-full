// Package middleware contains all Gin middleware for the application.
//
// In Node.js (Express), middleware is any function with signature:
//   (req, res, next) => { ... }
//
// In Gin (Go), middleware is any function that returns a gin.HandlerFunc:
//   func MyMiddleware() gin.HandlerFunc {
//       return func(c *gin.Context) {
//           // Before handler
//           c.Next()
//           // After handler
//       }
//   }
//
// The mental model is IDENTICAL. The key difference is that Gin uses
// c *gin.Context instead of (req, res, next). The context carries
// both the request AND response, plus any values you want to pass
// downstream.
package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// Recovery is a middleware that catches panics and converts them into 500 responses.
//
// In Node.js, an unhandled throw in middleware crashes the request.
// Express has a global error handler that catches synchronous throws.
// In Go, a panic() in a goroutine will CRASH THE ENTIRE PROCESS unless
// recover() is called in a deferred function in that same goroutine.
//
// Concept: Goroutines, Panic, and Recover
//   - panic() is Go's emergency stop. Use it ONLY for truly unrecoverable
//     situations (programming errors, not user input errors).
//   - recover() can catch a panic, but ONLY inside a deferred function.
//   - Every goroutine (including HTTP handler goroutines) needs its own
//     panic recovery, because panics don't cross goroutine boundaries.
//
// This middleware wraps every request handler with panic recovery.
func Recovery(isProduction bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// defer runs at function exit — even if panic() was called.
		defer func() {
			// recover() captures the panic value and stops the panic.
			// It returns nil if no panic occurred.
			if r := recover(); r != nil {
				// Log the full stack trace — invaluable for debugging.
				slog.Error("PANIC recovered",
					"panic", fmt.Sprintf("%v", r),
					"stack", string(debug.Stack()),
				)

				if isProduction {
					response.Error(c, 500, "An unexpected error occurred")
				} else {
					response.ErrorWithStack(c, 500,
						fmt.Sprintf("panic: %v", r),
						string(debug.Stack()),
					)
				}

				// c.Abort() stops the middleware chain — prevents any further handlers from running.
				// Without Abort(), Gin would continue to the next handler after the panic.
				c.Abort()
			}
		}()

		// c.Next() calls the next middleware/handler in the chain.
		// This is equivalent to calling next() in Express.
		c.Next()
	}
}

// ErrorHandler is the GLOBAL error handler middleware.
// It must be registered FIRST (before all routes) so it wraps the entire chain.
//
// In Node.js (Express), the global error handler has 4 parameters:
//   app.use((err, req, res, next) => { ... })  ← the extra 'err' marks it
//
// In Gin, there's no separate "error handler" position.
// Instead, handlers call c.Error(err) to register errors, and this
// middleware reads c.Errors after c.Next() returns.
//
// Alternatively (and this is what we do), handlers return early by calling
// c.JSON() + c.Abort() after checking for errors themselves, which is
// cleaner and matches how most real Gin apps work.
//
// For this project, we use a convention:
//   - Handlers call response.Error() or response.OK() and return
//   - This middleware catches any c.Errors that slipped through
//   - It also handles AppError type checking
func ErrorHandler(isProduction bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // let the handler run first

		// If c.Errors is non-empty, something went wrong.
		if len(c.Errors) == 0 {
			return
		}

		// Get the last error registered (the most specific one).
		err := c.Errors.Last().Err

		// Concept: errors.As — type-aware error inspection
		//   errors.As walks the error chain (via Unwrap()) looking for
		//   an error that can be assigned to the target type.
		//   This replaces instanceof checks in JavaScript:
		//     if (err instanceof AppError) { ... }
		//   becomes:
		//     var appErr *apperror.AppError
		//     if errors.As(err, &appErr) { ... }
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			if isProduction {
				response.Error(c, appErr.StatusCode, appErr.Message)
			} else {
				response.ErrorWithStack(c, appErr.StatusCode, appErr.Message, appErr.Error())
			}
			return
		}

		// Gemini rate limit errors → 503 (mirrors Node.js error.middleware.js)
		if isRateLimitError(err) {
			response.Error(c, 503, "The AI service is temporarily at its usage limit. Please try again in a few minutes.")
			return
		}

		// Fallback: unknown error → 500
		if isProduction {
			response.Error(c, 500, "An internal server error occurred")
		} else {
			response.ErrorWithStack(c, 500, err.Error(), fmt.Sprintf("%+v", err))
		}
	}
}

// isRateLimitError checks if an error is a Gemini API rate-limit error.
// Mirrors the Node.js: err.status === 429 || /quota|rate.?limit/i.test(err.message)
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// Simple substring checks — no regex needed for this case
	for _, keyword := range []string{"429", "quota", "rate limit", "rate-limit"} {
		for i := 0; i <= len(msg)-len(keyword); i++ {
			// case-insensitive check
			match := true
			for j := 0; j < len(keyword); j++ {
				c1, c2 := msg[i+j], keyword[j]
				if c1 != c2 && (c1 < 'A' || c1 > 'Z' || c1+32 != c2) && (c2 < 'A' || c2 > 'Z' || c2+32 != c1) {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

// NotFound handles 404 for unmatched routes.
// In Express:
//   app.use((req, res, next) => {
//     const error = new Error(`Not Found - ${req.originalUrl}`);
//     res.status(404); next(error);
//   })
//
// In Gin, we register this as gin.NoRoute():
//   r.NoRoute(middleware.NotFound())
func NotFound() gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Error(c, 404, fmt.Sprintf("Route not found: %s %s", c.Request.Method, c.Request.URL.Path))
	}
}

// Logger logs each HTTP request — equivalent to morgan('dev') in Node.js.
// We use Go 1.21+'s built-in log/slog (structured logging) instead of a
// third-party library. slog writes JSON in production, text in development.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // let the handler complete

		// slog.Info writes a structured log entry.
		// Structured logs have key-value pairs, not just a string.
		// This makes them parseable by tools like Datadog, CloudWatch, etc.
		slog.Info("HTTP request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"ip", c.ClientIP(),
		)
	}
}

