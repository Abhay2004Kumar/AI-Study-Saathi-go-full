// Package handler contains all Gin HTTP handlers for the application.
//
// In Node.js, controllers have this shape:
//   const register = async (req, res, next) => { ... }
//   module.exports = { register }
//
// In Go, handlers are methods on a struct that holds dependencies.
// This is Go's approach to dependency injection — no DI framework needed.
//
// The pattern:
//   type AuthHandler struct {
//       authService service.AuthService  // the dependency
//   }
//   func (h *AuthHandler) Register(c *gin.Context) { ... }
//
// Why struct-based handlers?
//   - Testing: you can inject a mock service
//   - Explicit dependencies: you know exactly what each handler needs
//   - No global state: each handler gets its deps via its struct, not globals
//
// In Node.js, you imported prisma and config globally:
//   const prisma = require('../config/db')
//   const config = require('../config/env')
// This creates hidden global dependencies that are hard to test and reason about.
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/middleware"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// AuthHandler holds all dependencies the auth routes need.
// Think of this as the Node.js controller + its imports bundled together.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler is the constructor — creates an AuthHandler with its dependencies.
// This is called once at startup when wiring everything together in main.go.
//
// Concept: Interfaces for Dependency Injection
//   authService is of type service.AuthService — an INTERFACE, not a concrete type.
//   This means AuthHandler doesn't care HOW the auth logic is implemented.
//   In tests, you pass a MockAuthService. In production, you pass the real one.
//   The handler code is IDENTICAL in both cases.
//
//   JavaScript equivalent (duck typing):
//     function newAuthHandler(authService) {
//       return { register: ..., login: ... }
//     }
//   Go equivalent (structural typing + compile-time check):
//     func NewAuthHandler(authService service.AuthService) *AuthHandler { ... }
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register handles POST /api/auth/register
//
// Node.js equivalent:
//   const register = async (req, res, next) => {
//     const { name, email, password } = req.body;
//     ...
//     res.status(201).json({ success: true, data: { id, name, email, token } });
//   }
//
// In Go:
//   - c.ShouldBindJSON(&req) replaces destructuring req.body AND Joi validation
//   - The binding:"required,min=2" tags on the DTO struct ARE the validation rules
//   - No separate validator middleware needed — Gin runs validation on ShouldBindJSON
func (h *AuthHandler) Register(c *gin.Context) {
	// --- Step 1: Parse and validate the request body ---
	//
	// Concept: ShouldBindJSON
	//   ShouldBindJSON reads the JSON body, unmarshals it into the struct,
	//   AND runs all the `binding:` validations defined on the struct fields.
	//   If any validation fails, it returns an error describing what failed.
	//
	//   var req dto.RegisterRequest declares a zero-value struct (all fields empty).
	//   We pass &req (a pointer) so ShouldBindJSON can WRITE INTO it.
	//   Without the &, Go would pass a copy, and changes wouldn't be visible here.
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return // return early — we're done with this request
	}

	// --- Step 2: Call the service ---
	//
	// We pass c.Request.Context() so the service can use it for DB queries.
	// If the HTTP request is cancelled (client disconnects), the context
	// will be cancelled too, and DB queries will abort automatically.
	result, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		// Concept: errors.As for type-safe error handling
		//   We check if the error is our custom AppError type.
		//   If it is, we use its StatusCode. Otherwise, it's an unexpected 500.
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.StatusCode, appErr.Message)
			return
		}
		slog.Error("Registration failed", "error", err)
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// --- Step 3: Send the success response ---
	response.Created(c, result)
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.StatusCode, appErr.Message)
			return
		}
		slog.Error("Login failed", "error", err)
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, result)
}

// GetMe handles GET /api/auth/me (protected)
func (h *AuthHandler) GetMe(c *gin.Context) {
	// middleware.GetUserID reads the userID that the Protect middleware stored in context.
	// This is how Go passes authenticated user info through the call chain
	// without touching req.user like Node.js does.
	userID := middleware.GetUserID(c.Request.Context())
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "Not authorized")
		return
	}

	user, err := h.authService.GetMe(c.Request.Context(), userID)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.StatusCode, appErr.Message)
			return
		}
		slog.Error("GetMe failed", "error", err)
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, user)
}

