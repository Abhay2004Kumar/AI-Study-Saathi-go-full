// Package service implements the business logic for Document management.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/google/uuid"
)

var allowedExtensions = map[string]bool{
	".pdf":  true,
	".txt":  true,
	".md":   true,
	".jpg":  true,
	".jpeg": true,
	".png":  true,
}

var validCategories = map[string]bool{
	"SYLLABUS": true,
	"NOTES":    true,
	"PYQ":      true,
	"GENERAL":  true,
}

// DocumentService defines all operations for Document management.
type DocumentService interface {
	Upload(ctx context.Context, userID, title, category string, examID *string, fileHeader *multipart.FileHeader) (*dto.DocumentSummary, error)
	List(ctx context.Context, userID string, examID *string) ([]dto.DocumentSummary, error)
	GetByID(ctx context.Context, userID, documentID string) (*dto.DocumentDetailResponse, error)
	UpdateCategory(ctx context.Context, userID, documentID, category string) (*dto.DocumentSummary, error)
	Delete(ctx context.Context, userID, documentID string) error
	GetStatus(ctx context.Context, userID, documentID string) (*dto.DocumentStatusResponse, error)
	Retry(ctx context.Context, userID, documentID string) (*dto.DocumentSummary, error)
}

type documentService struct {
	docRepo    repository.DocumentRepository
	examRepo   repository.ExamRepository
	uploadDir  string
	onEnqueued func(docID string)
}

// NewDocumentService creates a new DocumentService with background enqueue hook.
func NewDocumentService(
	docRepo repository.DocumentRepository,
	examRepo repository.ExamRepository,
	uploadDir string,
	onEnqueued func(docID string),
) DocumentService {
	_ = os.MkdirAll(uploadDir, 0755)
	return &documentService{
		docRepo:    docRepo,
		examRepo:   examRepo,
		uploadDir:  uploadDir,
		onEnqueued: onEnqueued,
	}
}

func (s *documentService) Upload(
	ctx context.Context,
	userID, title, category string,
	examID *string,
	fileHeader *multipart.FileHeader,
) (*dto.DocumentSummary, error) {
	// 1. Basic Validations
	if fileHeader == nil {
		return nil, apperror.BadRequest("Please upload a file")
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, apperror.BadRequest("Please provide a title for the document")
	}

	if category == "" {
		category = "GENERAL"
	} else if !validCategories[category] {
		return nil, apperror.BadRequest("Invalid category. Must be one of: SYLLABUS, NOTES, PYQ, GENERAL")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedExtensions[ext] {
		return nil, apperror.BadRequest("Invalid file type. Only PDF, TXT, Markdown, JPG, and PNG files are allowed.")
	}

	// 2. Validate Exam Ownership if linked
	if examID != nil && *examID != "" {
		exam, err := s.examRepo.FindByID(ctx, *examID, userID)
		if err != nil {
			return nil, fmt.Errorf("verifying exam: %w", err)
		}
		if exam == nil {
			return nil, apperror.NotFound("Exam not found")
		}
	}

	// 3. Open Uploaded File
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("opening uploaded file: %w", err)
	}
	defer src.Close()

	// 4. Prepare Destination File on Disk
	uniqueSuffix := fmt.Sprintf("%d-%s", os.Getpid(), uuid.New().String()[:8])
	targetFileName := fmt.Sprintf("file-%d-%s%s", os.Getpid(), uniqueSuffix, ext)
	targetFilePath := filepath.Join(s.uploadDir, targetFileName)

	dst, err := os.Create(targetFilePath)
	if err != nil {
		return nil, fmt.Errorf("creating destination file: %w", err)
	}

	// Flag to track whether we should delete the file if DB save fails
	success := false
	defer func() {
		dst.Close()
		if !success {
			_ = os.Remove(targetFilePath) // clean up partial or orphaned file
		}
	}()

	// 5. Stream from source to destination WHILE hashing with SHA-256 (TeeReader)
	hasher := sha256.New()
	tee := io.TeeReader(src, hasher)

	fileSize, err := io.Copy(dst, tee)
	if err != nil {
		return nil, fmt.Errorf("writing file to disk: %w", err)
	}

	fileHash := hex.EncodeToString(hasher.Sum(nil))

	// 6. Check for duplicate file for this user
	dup, err := s.docRepo.FindByHash(ctx, userID, fileHash)
	if err != nil {
		return nil, fmt.Errorf("checking duplicate: %w", err)
	}
	if dup != nil {
		return nil, apperror.Conflict(fmt.Sprintf("This exact file was already uploaded as %q", dup.Title))
	}

	// 7. Store Document Record in Database
	docID := uuid.New().String()
	doc := &model.Document{
		ID:               docID,
		UserID:           userID,
		ExamID:           examID,
		Title:            title,
		FileName:         fileHeader.Filename,
		FilePath:         targetFilePath,
		FileType:         fileHeader.Header.Get("Content-Type"),
		FileSize:         fileSize,
		FileHash:         &fileHash,
		Category:         category,
		ProcessingStatus: "UPLOADED",
	}

	if doc.FileType == "" {
		doc.FileType = "application/octet-stream"
	}

	if err := s.docRepo.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("saving document record: %w", err)
	}

	// Success! Do not unlink the file
	success = true

	// Enqueue for background chunking & vector embedding
	if s.onEnqueued != nil {
		s.onEnqueued(doc.ID)
	}

	return toDocumentSummary(doc), nil
}

func (s *documentService) List(ctx context.Context, userID string, examID *string) ([]dto.DocumentSummary, error) {
	docs, err := s.docRepo.FindAllByUser(ctx, userID, examID)
	if err != nil {
		return nil, fmt.Errorf("listing documents: %w", err)
	}

	summaries := make([]dto.DocumentSummary, len(docs))
	for i, d := range docs {
		summaries[i] = *toDocumentSummary(&d)
	}
	return summaries, nil
}

func (s *documentService) GetByID(ctx context.Context, userID, documentID string) (*dto.DocumentDetailResponse, error) {
	doc, err := s.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("finding document: %w", err)
	}
	if doc == nil {
		return nil, apperror.NotFound("Document not found")
	}
	if doc.UserID != userID {
		return nil, apperror.Forbidden("Not authorized to access this document")
	}

	return &dto.DocumentDetailResponse{
		ID:                       doc.ID,
		UserID:                   doc.UserID,
		ExamID:                   doc.ExamID,
		Title:                    doc.Title,
		FileName:                 doc.FileName,
		FileType:                 doc.FileType,
		FileSize:                 doc.FileSize,
		Category:                 doc.Category,
		AICategory:               doc.AICategory,
		ClassificationConfidence: doc.ClassificationConfidence,
		PageCount:                doc.PageCount,
		ProcessingStatus:         doc.ProcessingStatus,
		CreatedAt:                doc.CreatedAt,
		UpdatedAt:                doc.UpdatedAt,
	}, nil
}

func (s *documentService) UpdateCategory(ctx context.Context, userID, documentID, category string) (*dto.DocumentSummary, error) {
	if !validCategories[category] {
		return nil, apperror.BadRequest("Invalid category. Must be one of: SYLLABUS, NOTES, PYQ, GENERAL")
	}

	doc, err := s.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if doc == nil || doc.UserID != userID {
		return nil, apperror.NotFound("Document not found")
	}

	updated, err := s.docRepo.Update(ctx, documentID, map[string]interface{}{"category": category})
	if err != nil {
		return nil, fmt.Errorf("updating category: %w", err)
	}

	return toDocumentSummary(updated), nil
}

func (s *documentService) Delete(ctx context.Context, userID, documentID string) error {
	doc, err := s.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return err
	}
	if doc == nil || doc.UserID != userID {
		return apperror.NotFound("Document not found")
	}

	// 1. Delete from database
	if err := s.docRepo.Delete(ctx, documentID); err != nil {
		return fmt.Errorf("deleting document row: %w", err)
	}

	// 2. Unlink from filesystem
	if doc.FilePath != "" {
		_ = os.Remove(doc.FilePath)
	}

	return nil
}

func (s *documentService) GetStatus(ctx context.Context, userID, documentID string) (*dto.DocumentStatusResponse, error) {
	doc, err := s.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if doc == nil || doc.UserID != userID {
		return nil, apperror.NotFound("Document not found")
	}

	return &dto.DocumentStatusResponse{Status: doc.ProcessingStatus}, nil
}

func (s *documentService) Retry(ctx context.Context, userID, documentID string) (*dto.DocumentSummary, error) {
	doc, err := s.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if doc == nil || doc.UserID != userID {
		return nil, apperror.NotFound("Document not found")
	}

	if doc.ProcessingStatus != "FAILED" {
		return nil, apperror.BadRequest(fmt.Sprintf("Only failed documents can be retried (current status: %s)", doc.ProcessingStatus))
	}

	updated, err := s.docRepo.Update(ctx, documentID, map[string]interface{}{"processingStatus": "UPLOADED"})
	if err != nil {
		return nil, fmt.Errorf("resetting status to UPLOADED: %w", err)
	}

	if s.onEnqueued != nil {
		s.onEnqueued(documentID)
	}

	return toDocumentSummary(updated), nil
}

func toDocumentSummary(d *model.Document) *dto.DocumentSummary {
	return &dto.DocumentSummary{
		ID:                       d.ID,
		Title:                    d.Title,
		FileName:                 d.FileName,
		FileType:                 d.FileType,
		FileSize:                 d.FileSize,
		ExamID:                   d.ExamID,
		Category:                 d.Category,
		AICategory:               d.AICategory,
		ClassificationConfidence: d.ClassificationConfidence,
		PageCount:                d.PageCount,
		ProcessingStatus:         d.ProcessingStatus,
		CreatedAt:                d.CreatedAt,
		UpdatedAt:                d.UpdatedAt,
	}
}

