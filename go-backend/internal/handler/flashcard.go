// Package handler provides HTTP endpoints for flashcard management and SM-2 reviews.
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/middleware"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// FlashcardHandler handles flashcard-related HTTP endpoints.
type FlashcardHandler struct {
	cardService service.FlashcardService
}

// NewFlashcardHandler creates a new FlashcardHandler.
func NewFlashcardHandler(cardService service.FlashcardService) *FlashcardHandler {
	return &FlashcardHandler{cardService: cardService}
}

// GenerateTopicFlashcards handles POST /api/exams/:id/topics/:topicId/flashcards
func (h *FlashcardHandler) GenerateTopicFlashcards(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")
	topicID := c.Param("topicId")

	var req dto.GenerateFlashcardsRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	cards, err := h.cardService.GenerateTopicFlashcards(c.Request.Context(), userID, examID, topicID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, cards)
}

// ListTopicFlashcards handles GET /api/exams/:id/topics/:topicId/flashcards
func (h *FlashcardHandler) ListTopicFlashcards(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")
	topicID := c.Param("topicId")

	cards, err := h.cardService.ListTopicFlashcards(c.Request.Context(), userID, examID, topicID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, cards)
}

// GetDueFlashcards handles GET /api/exams/:id/flashcards/due
func (h *FlashcardHandler) GetDueFlashcards(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	cards, err := h.cardService.GetDueFlashcards(c.Request.Context(), userID, examID, limit)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, cards)
}

// ReviewFlashcard handles POST /api/flashcards/:id/review
func (h *FlashcardHandler) ReviewFlashcard(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	cardID := c.Param("id")

	var req dto.ReviewFlashcardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "grade must be one of: AGAIN, HARD, GOOD, EASY")
		return
	}

	updated, err := h.cardService.ReviewFlashcard(c.Request.Context(), userID, cardID, req.Grade)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, updated)
}

func (h *FlashcardHandler) handleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.StatusCode, appErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, err.Error())
}

