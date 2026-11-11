// Package handler provides HTTP endpoints for Socratic Tutoring.
package handler

import (
	"errors"
	"net/http"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/middleware"
	"github.com/aiStudySaathi/backend/internal/service/ai"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// TutoringHandler handles tutoring endpoints.
type TutoringHandler struct {
	tutorService ai.TutoringService
}

// NewTutoringHandler creates a new TutoringHandler.
func NewTutoringHandler(tutorService ai.TutoringService) *TutoringHandler {
	return &TutoringHandler{tutorService: tutorService}
}

// StartSession handles POST /api/tutoring/session/start
func (h *TutoringHandler) StartSession(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())

	var req dto.StartTutoringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Subject and topic are required")
		return
	}

	result, err := h.tutorService.StartSession(c.Request.Context(), userID, req.Subject, req.Topic)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, result)
}

// RespondToTutor handles POST /api/tutoring/session/:id/respond
func (h *TutoringHandler) RespondToTutor(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	sessionID := c.Param("id")

	var req dto.RespondToTutorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Answer is required")
		return
	}

	result, err := h.tutorService.Respond(c.Request.Context(), userID, sessionID, req.Answer)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

// GetSession handles GET /api/tutoring/session/:id
func (h *TutoringHandler) GetSession(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	sessionID := c.Param("id")

	session, err := h.tutorService.GetSession(c.Request.Context(), userID, sessionID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, session)
}

func (h *TutoringHandler) handleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.StatusCode, appErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, err.Error())
}

