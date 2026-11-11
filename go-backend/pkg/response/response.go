// Package response provides standardised JSON response helpers for Gin handlers.
//
// In Node.js, you wrote responses directly in controllers:
//   res.status(201).json({ success: true, data: user })
//   res.status(400).json({ success: false, message: 'Bad request' })
//
// In Go with Gin, this package enforces a consistent response shape across the entire API —
// exactly matching what the Node.js backend produces so frontend clients work without modification.
//
// Every API response has the shape:
//   { "success": true, "data": {...} }                // success with data (or "data": null)
//   { "success": true, "message": "..." }              // success with message only (e.g. delete)
//   { "success": false, "message": "..." }             // error
//   { "success": false, "message": "...", "stack": "" }// error in dev
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SuccessResponse represents a successful JSON response with a required data field.
// Note: Data does NOT have omitempty so that nil serializes as `data: null`,
// guaranteeing parity with Node.js res.json({ success: true, data: null }).
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// MessageResponse represents a successful JSON response without a data payload.
type MessageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Stack   string `json:"stack,omitempty"`
}

// APIResponse is retained for backward-compatibility.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
	Stack   string      `json:"stack,omitempty"`
}

// OK sends a 200 success response with data.
// Equivalent to Node.js: res.json({ success: true, data })
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// Created sends a 201 success response with data.
// Equivalent to Node.js: res.status(201).json({ success: true, data })
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// NoContent sends a 200 success response with just a message.
// Equivalent to Node.js: res.json({ success: true, message })
func NoContent(c *gin.Context, message string) {
	c.JSON(http.StatusOK, MessageResponse{
		Success: true,
		Message: message,
	})
}

// Error sends an error response.
// Equivalent to Node.js: res.status(statusCode).json({ success: false, message })
func Error(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Message: message,
	})
}

// ErrorWithStack sends an error response with a stack trace (dev mode).
func ErrorWithStack(c *gin.Context, statusCode int, message, stack string) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Message: message,
		Stack:   stack,
	})
}
