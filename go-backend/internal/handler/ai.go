// Package handler implements AI-related endpoints (RAG Q&A).
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

// AIHandler handles /api/ai routes.
type AIHandler struct {
	ragService ai.RAGService
}

// NewAIHandler creates a new AIHandler.
func NewAIHandler(ragService ai.RAGService) *AIHandler {
	return &AIHandler{ragService: ragService}
}

// AskQuestion handles POST /api/ai/ask
func (h *AIHandler) AskQuestion(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())

	var req dto.AskQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	var examIDPtr *string
	if req.ExamID != "" {
		examIDPtr = &req.ExamID
	}
	var categoryPtr *string
	if req.Category != "" {
		categoryPtr = &req.Category
	}

	result, err := h.ragService.AskQuestion(c.Request.Context(), userID, req.Question, ai.RAGFilters{
		ExamID:   examIDPtr,
		Category: categoryPtr,
	})
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.StatusCode, appErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, result)
}

