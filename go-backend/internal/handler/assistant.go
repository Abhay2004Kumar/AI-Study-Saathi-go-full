// Package handler provides HTTP endpoints for the AI Study Assistant.
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

// AssistantHandler handles AI Assistant routes.
type AssistantHandler struct {
	assistantService ai.AssistantService
}

// NewAssistantHandler creates a new AssistantHandler.
func NewAssistantHandler(assistantService ai.AssistantService) *AssistantHandler {
	return &AssistantHandler{assistantService: assistantService}
}

// GetAssistantSession handles GET /api/exams/:id/assistant
func (h *AssistantHandler) GetAssistantSession(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")

	session, err := h.assistantService.GetSession(c.Request.Context(), userID, examID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, session)
}

// SendAssistantMessage handles POST /api/exams/:id/assistant/message
func (h *AssistantHandler) SendAssistantMessage(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")

	var req dto.SendAssistantMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "message is required")
		return
	}

	result, err := h.assistantService.SendMessage(c.Request.Context(), userID, examID, req.Message)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, result)
}

func (h *AssistantHandler) handleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.StatusCode, appErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, err.Error())
}

