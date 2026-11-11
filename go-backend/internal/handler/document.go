// Package handler implements HTTP endpoints for Document management.
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

// DocumentHandler handles /api/documents routes.
type DocumentHandler struct {
	docService service.DocumentService
}

// NewDocumentHandler creates a new DocumentHandler.
func NewDocumentHandler(docService service.DocumentService) *DocumentHandler {
	return &DocumentHandler{docService: docService}
}

// Upload handles POST /api/documents (multipart/form-data)
func (h *DocumentHandler) Upload(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())

	// 1. Read file from multipart form
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Please upload a file")
		return
	}

	// 2. Read other form fields
	title := c.PostForm("title")
	category := c.PostForm("category")
	var examIDPtr *string
	if examID := c.PostForm("examId"); examID != "" {
		examIDPtr = &examID
	}

	// 3. Delegate to service
	doc, err := h.docService.Upload(c.Request.Context(), userID, title, category, examIDPtr, fileHeader)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, doc)
}

// List handles GET /api/documents
func (h *DocumentHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())

	var examIDPtr *string
	if examID := c.Query("examId"); examID != "" {
		examIDPtr = &examID
	}

	docs, err := h.docService.List(c.Request.Context(), userID, examIDPtr)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, docs)
}

// GetByID handles GET /api/documents/:id
func (h *DocumentHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	docID := c.Param("id")

	doc, err := h.docService.GetByID(c.Request.Context(), userID, docID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, doc)
}

// Update handles PATCH /api/documents/:id
func (h *DocumentHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	docID := c.Param("id")

	var req dto.UpdateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.docService.UpdateCategory(c.Request.Context(), userID, docID, req.Category)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, updated)
}

// Delete handles DELETE /api/documents/:id
func (h *DocumentHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	docID := c.Param("id")

	err := h.docService.Delete(c.Request.Context(), userID, docID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.NoContent(c, "Document deleted successfully")
}

// GetStatus handles GET /api/documents/:id/status
func (h *DocumentHandler) GetStatus(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	docID := c.Param("id")

	status, err := h.docService.GetStatus(c.Request.Context(), userID, docID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, status)
}

// Retry handles POST /api/documents/:id/retry
func (h *DocumentHandler) Retry(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())
	docID := c.Param("id")

	retried, err := h.docService.Retry(c.Request.Context(), userID, docID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, retried)
}

func (h *DocumentHandler) handleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.StatusCode, appErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, err.Error())
}

