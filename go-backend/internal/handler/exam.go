// Package handler implements HTTP routing endpoints for Exam management.
package handler

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/middleware"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/internal/service/ai"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/gemini"
	"github.com/aiStudySaathi/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// ExamHandler handles all requests for /api/exams
type ExamHandler struct {
	examService  service.ExamService
	geminiClient *gemini.Client
}

// NewExamHandler constructor
func NewExamHandler(examService service.ExamService, geminiClient ...*gemini.Client) *ExamHandler {
	var client *gemini.Client
	if len(geminiClient) > 0 {
		client = geminiClient[0]
	}
	return &ExamHandler{examService: examService, geminiClient: client}
}

// Create handles POST /api/exams
func (h *ExamHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())

	bodyBytes, _ := c.GetRawData()
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req dto.CreateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("CreateExam binding failed", "error", err, "body", string(bodyBytes))
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.examService.CreateExam(c.Request.Context(), userID, req)
	if err != nil {
		slog.Warn("CreateExam service failed", "error", err, "body", string(bodyBytes))
		h.handleError(c, err)
		return
	}

	response.Created(c, result)
}

// List handles GET /api/exams
func (h *ExamHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())

	exams, err := h.examService.ListExams(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, exams)
}

// GetByID handles GET /api/exams/:id
func (h *ExamHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")

	exam, err := h.examService.GetExamByID(c.Request.Context(), userID, examID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, exam)
}

// Update handles PATCH /api/exams/:id
func (h *ExamHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")

	var req dto.UpdateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	exam, err := h.examService.UpdateExam(c.Request.Context(), userID, examID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, exam)
}

// Delete handles DELETE /api/exams/:id
func (h *ExamHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")

	err := h.examService.DeleteExam(c.Request.Context(), userID, examID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.NoContent(c, "Exam deleted successfully")
}

// ReplaceSyllabus handles PUT /api/exams/:id/syllabus
func (h *ExamHandler) ReplaceSyllabus(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	examID := c.Param("id")

	bodyBytes, _ := c.GetRawData()
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req dto.ReplaceSyllabusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("ReplaceSyllabus binding failed", "error", err, "body", string(bodyBytes))
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.examService.ReplaceSyllabus(c.Request.Context(), userID, examID, req.Subjects)
	if err != nil {
		slog.Warn("ReplaceSyllabus service failed", "error", err, "body", string(bodyBytes))
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

// ExtractSyllabus handles POST /api/exams/syllabus/extract
// Accepts either a JSON body { "text": "..." } or multipart form with file (PDF, TXT, MD) and/or text field.
func (h *ExamHandler) ExtractSyllabus(c *gin.Context) {
	var rawText string

	// 1. Check for multipart file upload
	file, err := c.FormFile("file")
	if err == nil && file != nil {
		tempFile, tempErr := os.CreateTemp("", "syllabus-*"+filepath.Ext(file.Filename))
		if tempErr == nil {
			tempPath := tempFile.Name()
			tempFile.Close()
			defer os.Remove(tempPath)

			if saveErr := c.SaveUploadedFile(file, tempPath); saveErr == nil {
				if text, extractErr := ai.ExtractTextFromFile(c.Request.Context(), tempPath, h.geminiClient); extractErr == nil {
					rawText = text
				}
			}
		}
	}

	// 2. Fall back to form field "text" if no file text
	if strings.TrimSpace(rawText) == "" {
		rawText = c.PostForm("text")
	}

	// 3. Fall back to JSON body if not form
	if strings.TrimSpace(rawText) == "" {
		var req dto.ExtractSyllabusRequest
		if bindErr := c.ShouldBindJSON(&req); bindErr == nil {
			rawText = req.Text
		}
	}

	if strings.TrimSpace(rawText) == "" {
		response.Error(c, http.StatusBadRequest, "Provide syllabus text or upload a file to extract from")
		return
	}

	result, err := h.examService.ExtractSyllabus(c.Request.Context(), rawText)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

func (h *ExamHandler) handleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.StatusCode, appErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, err.Error())
}

