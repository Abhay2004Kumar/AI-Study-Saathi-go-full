// Package handler provides HTTP endpoints for PYQ analysis and status updates.
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

// PyqHandler handles PYQ-related endpoints.
type PyqHandler struct {
	pyqService service.PyqService
}

// NewPyqHandler creates a new PyqHandler.
func NewPyqHandler(pyqService service.PyqService) *PyqHandler {
	return &PyqHandler{pyqService: pyqService}
}

// GetTopicPyqAnalysis handles GET /api/exams/:id/topics/:topicId/pyq-analysis
func (h *PyqHandler) GetTopicPyqAnalysis(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")
	topicID := c.Param("topicId")

	result, err := h.pyqService.GetTopicPyqAnalysis(c.Request.Context(), userID, examID, topicID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

// UpdateQuestionStatus handles PATCH /api/pyq-questions/:questionId
func (h *PyqHandler) UpdateQuestionStatus(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	questionID := c.Param("questionId")

	var req dto.UpdateQuestionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Status must be one of: CONFIRMED, REJECTED, PENDING_REVIEW")
		return
	}

	result, err := h.pyqService.UpdateQuestionStatus(c.Request.Context(), userID, questionID, req.Status)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

// GetDocumentQuestions handles GET /api/documents/:id/questions
func (h *PyqHandler) GetDocumentQuestions(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	documentID := c.Param("id")

	result, err := h.pyqService.GetDocumentQuestions(c.Request.Context(), userID, documentID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

// AnalyzePyqDocument handles POST /api/documents/:id/analyze-pyq
func (h *PyqHandler) AnalyzePyqDocument(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	documentID := c.Param("id")

	result, err := h.pyqService.AnalyzePyqDocument(c.Request.Context(), userID, documentID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

func (h *PyqHandler) handleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.StatusCode, appErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, err.Error())
}

