// Package handler provides HTTP endpoints for Adaptive Study Plans and Session tracking.
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

// StudyPlanHandler handles study plan endpoints.
type StudyPlanHandler struct {
	planService service.StudyPlanService
}

// NewStudyPlanHandler creates a new StudyPlanHandler.
func NewStudyPlanHandler(planService service.StudyPlanService) *StudyPlanHandler {
	return &StudyPlanHandler{planService: planService}
}

// GenerateStudyPlan handles POST /api/exams/:id/study-plan
func (h *StudyPlanHandler) GenerateStudyPlan(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")

	result, err := h.planService.Generate(c.Request.Context(), userID, examID, nil)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, result)
}

// GetStudyPlan handles GET /api/exams/:id/study-plan
func (h *StudyPlanHandler) GetStudyPlan(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")

	plan, err := h.planService.GetActivePlan(c.Request.Context(), userID, examID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, plan)
}

// UpdateSessionCompletion handles PATCH /api/study-plan-sessions/:sessionId
func (h *StudyPlanHandler) UpdateSessionCompletion(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	sessionID := c.Param("sessionId")

	var req dto.UpdateSessionCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "completed must be a boolean")
		return
	}

	session, err := h.planService.SetSessionCompletion(c.Request.Context(), userID, sessionID, *req.Completed)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, session)
}

func (h *StudyPlanHandler) handleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		slog.Warn("StudyPlan client error", "status", appErr.StatusCode, "message", appErr.Message, "path", c.Request.URL.Path)
		response.Error(c, appErr.StatusCode, appErr.Message)
		return
	}
	slog.Error("StudyPlan internal error", "error", err, "path", c.Request.URL.Path, "examID", c.Param("id"))
	response.Error(c, http.StatusInternalServerError, err.Error())
}

