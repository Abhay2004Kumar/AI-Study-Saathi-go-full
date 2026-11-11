// Package handler provides HTTP endpoints for Mastery and PYQ intelligence.
package handler

import (
	"errors"
	"net/http"

	"github.com/aiStudySaathi/backend/internal/middleware"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// MasteryHandler handles student mastery endpoints.
type MasteryHandler struct {
	masteryService service.MasteryService
}

// NewMasteryHandler creates a new MasteryHandler.
func NewMasteryHandler(masteryService service.MasteryService) *MasteryHandler {
	return &MasteryHandler{masteryService: masteryService}
}

// GetTopicMastery handles GET /api/exams/:id/topics/:topicId/mastery
func (h *MasteryHandler) GetTopicMastery(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")
	topicID := c.Param("topicId")

	result, err := h.masteryService.GetTopicMastery(c.Request.Context(), userID, examID, topicID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

// GetExamMastery handles GET /api/exams/:id/mastery
func (h *MasteryHandler) GetExamMastery(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")

	result, err := h.masteryService.GetExamMastery(c.Request.Context(), userID, examID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

func (h *MasteryHandler) handleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.StatusCode, appErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, err.Error())
}

