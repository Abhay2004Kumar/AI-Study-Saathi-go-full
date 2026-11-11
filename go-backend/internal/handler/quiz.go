// Package handler provides HTTP request handling for Quizzes and Attempts.
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

// QuizHandler handles quiz-related HTTP endpoints.
type QuizHandler struct {
	quizService service.QuizService
}

// NewQuizHandler creates a new QuizHandler.
func NewQuizHandler(quizService service.QuizService) *QuizHandler {
	return &QuizHandler{quizService: quizService}
}

// GenerateTopicQuiz handles POST /api/exams/:id/topics/:topicId/quizzes
func (h *QuizHandler) GenerateTopicQuiz(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")
	topicID := c.Param("topicId")

	var req dto.GenerateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Difficulty != "" && req.Difficulty != "easy" && req.Difficulty != "medium" && req.Difficulty != "hard" {
		response.Error(c, http.StatusBadRequest, "difficulty must be one of: easy, medium, hard")
		return
	}

	result, err := h.quizService.GenerateTopicQuiz(c.Request.Context(), userID, examID, topicID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, result)
}

// ListTopicQuizzes handles GET /api/exams/:id/topics/:topicId/quizzes
func (h *QuizHandler) ListTopicQuizzes(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")
	topicID := c.Param("topicId")

	result, err := h.quizService.ListTopicQuizzes(c.Request.Context(), userID, examID, topicID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

// GetQuiz handles GET /api/quizzes/:quizId
func (h *QuizHandler) GetQuiz(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	quizID := c.Param("quizId")

	result, err := h.quizService.GetTakeableQuiz(c.Request.Context(), userID, quizID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

// SubmitAttempt handles POST /api/quizzes/:quizId/submit
func (h *QuizHandler) SubmitAttempt(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	quizID := c.Param("quizId")

	var req dto.SubmitQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "answers must be an array of { questionId, selectedAnswer }")
		return
	}

	result, err := h.quizService.SubmitAttempt(c.Request.Context(), userID, quizID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, result)
}

// ListQuizAttempts handles GET /api/quizzes/:quizId/attempts
func (h *QuizHandler) ListQuizAttempts(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	quizID := c.Param("quizId")

	result, err := h.quizService.ListQuizAttempts(c.Request.Context(), userID, quizID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

// GetQuizAttempt handles GET /api/quizzes/:quizId/attempts/:attemptId
func (h *QuizHandler) GetQuizAttempt(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	quizID := c.Param("quizId")
	attemptID := c.Param("attemptId")

	result, err := h.quizService.GetQuizAttempt(c.Request.Context(), userID, quizID, attemptID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

func (h *QuizHandler) handleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.StatusCode, appErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, err.Error())
}

