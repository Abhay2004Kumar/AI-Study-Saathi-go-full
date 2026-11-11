package service_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/pkg/apperror"
)

// MockDocumentRepository is an in-memory mock for DocumentRepository
type MockDocumentRepository struct {
	docsByID   map[string]*model.Document
	docsByHash map[string]*model.Document
}

func NewMockDocumentRepository() *MockDocumentRepository {
	return &MockDocumentRepository{
		docsByID:   make(map[string]*model.Document),
		docsByHash: make(map[string]*model.Document),
	}
}

func (m *MockDocumentRepository) Create(ctx context.Context, doc *model.Document) error {
	m.docsByID[doc.ID] = doc
	if doc.FileHash != nil {
		m.docsByHash[doc.UserID+":"+*doc.FileHash] = doc
	}
	return nil
}

func (m *MockDocumentRepository) FindByID(ctx context.Context, id string) (*model.Document, error) {
	return m.docsByID[id], nil
}

func (m *MockDocumentRepository) FindAllByUser(ctx context.Context, userID string, examID *string) ([]model.Document, error) {
	var list []model.Document
	for _, d := range m.docsByID {
		if d.UserID == userID {
			if examID != nil && (d.ExamID == nil || *d.ExamID != *examID) {
				continue
			}
			list = append(list, *d)
		}
	}
	return list, nil
}

func (m *MockDocumentRepository) FindByHash(ctx context.Context, userID, hash string) (*model.Document, error) {
	return m.docsByHash[userID+":"+hash], nil
}

func (m *MockDocumentRepository) UpdateStatus(ctx context.Context, id, status string) error {
	if doc, ok := m.docsByID[id]; ok {
		doc.ProcessingStatus = status
	}
	return nil
}

func (m *MockDocumentRepository) UpdatePageCount(ctx context.Context, id string, count int) error {
	if doc, ok := m.docsByID[id]; ok {
		doc.PageCount = &count
	}
	return nil
}

func (m *MockDocumentRepository) Update(ctx context.Context, id string, updates map[string]interface{}) (*model.Document, error) {
	doc, ok := m.docsByID[id]
	if !ok {
		return nil, nil
	}
	if cat, ok := updates["category"].(string); ok {
		doc.Category = cat
	}
	if status, ok := updates["processingStatus"].(string); ok {
		doc.ProcessingStatus = status
	}
	return doc, nil
}

func (m *MockDocumentRepository) Delete(ctx context.Context, id string) error {
	doc, ok := m.docsByID[id]
	if !ok {
		return apperror.NotFound("Document not found")
	}
	delete(m.docsByID, id)
	if doc.FileHash != nil {
		delete(m.docsByHash, doc.UserID+":"+*doc.FileHash)
	}
	return nil
}

// Helper to create a fake multipart.FileHeader in memory for unit testing
func createTestFileHeader(filename, content string) (*multipart.FileHeader, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", "text/plain")

	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 << 20)
	if err != nil {
		return nil, err
	}

	return form.File["file"][0], nil
}

func TestUploadDocumentSuccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "doc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	mockDocRepo := NewMockDocumentRepository()
	mockExamRepo := NewMockExamRepository()
	svc := service.NewDocumentService(mockDocRepo, mockExamRepo, tempDir, nil)

	fileHeader, err := createTestFileHeader("notes.txt", "These are operating system notes on semaphores.")
	if err != nil {
		t.Fatal(err)
	}

	summary, err := svc.Upload(context.Background(), "user-1", "OS Notes", "NOTES", nil, fileHeader)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if summary.Title != "OS Notes" {
		t.Errorf("expected title 'OS Notes', got: %s", summary.Title)
	}
	if summary.Category != "NOTES" {
		t.Errorf("expected category 'NOTES', got: %s", summary.Category)
	}
	if summary.ProcessingStatus != "UPLOADED" {
		t.Errorf("expected status 'UPLOADED', got: %s", summary.ProcessingStatus)
	}

	// Verify file was physically written to test directory
	files, _ := os.ReadDir(tempDir)
	if len(files) != 1 {
		t.Errorf("expected 1 file in upload directory, found: %d", len(files))
	}
}

func TestUploadDuplicateFile(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "doc-test-*")
	defer os.RemoveAll(tempDir)

	mockDocRepo := NewMockDocumentRepository()
	mockExamRepo := NewMockExamRepository()
	svc := service.NewDocumentService(mockDocRepo, mockExamRepo, tempDir, nil)

	content := "Unique content for duplicate check"
	file1, _ := createTestFileHeader("lecture1.txt", content)
	file2, _ := createTestFileHeader("lecture1_copy.txt", content)

	_, err := svc.Upload(context.Background(), "user-1", "Lecture 1", "NOTES", nil, file1)
	if err != nil {
		t.Fatalf("first upload failed: %v", err)
	}

	// Second upload with identical content must return 409 Conflict
	_, err = svc.Upload(context.Background(), "user-1", "Lecture 1 Copy", "NOTES", nil, file2)
	if err == nil {
		t.Fatal("expected duplicate upload error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.StatusCode != 409 {
		t.Errorf("expected 409 Conflict, got: %v", err)
	}
}

func TestUploadInvalidExtension(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "doc-test-*")
	defer os.RemoveAll(tempDir)

	mockDocRepo := NewMockDocumentRepository()
	mockExamRepo := NewMockExamRepository()
	svc := service.NewDocumentService(mockDocRepo, mockExamRepo, tempDir, nil)

	file, _ := createTestFileHeader("script.exe", "binary content")
	_, err := svc.Upload(context.Background(), "user-1", "Malicious File", "GENERAL", nil, file)
	if err == nil {
		t.Fatal("expected invalid extension error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.StatusCode != 400 {
		t.Errorf("expected 400 BadRequest, got: %v", err)
	}
}

func TestDeleteDocument(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "doc-test-*")
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "sample.txt")
	_ = os.WriteFile(filePath, []byte("hello"), 0644)

	mockDocRepo := NewMockDocumentRepository()
	mockExamRepo := NewMockExamRepository()
	svc := service.NewDocumentService(mockDocRepo, mockExamRepo, tempDir, nil)

	doc := &model.Document{
		ID:       "doc-123",
		UserID:   "user-1",
		Title:    "Sample",
		FilePath: filePath,
	}
	_ = mockDocRepo.Create(context.Background(), doc)

	err := svc.Delete(context.Background(), "user-1", "doc-123")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Verify file was unlinked from disk
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("expected physical file to be removed from disk, but it still exists")
	}
}

