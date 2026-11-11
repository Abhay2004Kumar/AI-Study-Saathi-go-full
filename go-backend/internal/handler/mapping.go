// Package handler provides HTTP endpoints for Resource Topic Mapping.
package handler

import (
	"errors"
	"net/http"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/middleware"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// MappingHandler handles resource topic mapping endpoints.
type MappingHandler struct {
	mappingService service.TopicMappingService
}

// NewMappingHandler creates a new MappingHandler.
func NewMappingHandler(mappingService service.TopicMappingService) *MappingHandler {
	return &MappingHandler{mappingService: mappingService}
}

// GetTopicResources handles GET /api/exams/:id/topics/:topicId/resources
func (h *MappingHandler) GetTopicResources(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")
	topicID := c.Param("topicId")

	result, err := h.mappingService.GetTopicResources(c.Request.Context(), userID, examID, topicID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

// GetDocumentMappings handles GET /api/documents/:id/mappings
func (h *MappingHandler) GetDocumentMappings(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	documentID := c.Param("id")

	result, err := h.mappingService.GetDocumentMappings(c.Request.Context(), userID, documentID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

// MapDocumentTopics handles POST /api/documents/:id/map-topics
func (h *MappingHandler) MapDocumentTopics(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	documentID := c.Param("id")

	result, err := h.mappingService.MapDocumentTopics(c.Request.Context(), userID, documentID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

// UpdateMappingStatus handles PATCH /api/mappings/:mappingId
func (h *MappingHandler) UpdateMappingStatus(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	mappingID := c.Param("mappingId")

	var req dto.UpdateMappingStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Status must be one of: CONFIRMED, REJECTED, PENDING_REVIEW")
		return
	}

	result, err := h.mappingService.UpdateMappingStatus(c.Request.Context(), userID, mappingID, req.Status)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

func (h *MappingHandler) handleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.StatusCode, appErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, err.Error())
}

