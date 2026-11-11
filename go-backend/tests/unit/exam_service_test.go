package service_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/internal/service/ai"
	"github.com/aiStudySaathi/backend/pkg/apperror"
)

// MockExamRepository is an in-memory mock for ExamRepository
type MockExamRepository struct {
	examsByID map[string]*model.Exam
}

func NewMockExamRepository() *MockExamRepository {
	return &MockExamRepository{
		examsByID: make(map[string]*model.Exam),
	}
}

func (m *MockExamRepository) Create(ctx context.Context, exam *model.Exam, subjects []model.Subject) (*model.Exam, error) {
	exam.ID = "test-exam-1"
	m.examsByID[exam.ID] = exam
	return exam, nil
}

func (m *MockExamRepository) FindByID(ctx context.Context, id, userID string) (*model.Exam, error) {
	exam, ok := m.examsByID[id]
	if !ok || exam.UserID != userID {
		return nil, nil
	}
	return exam, nil
}

func (m *MockExamRepository) FindAllByUser(ctx context.Context, userID string) ([]model.Exam, error) {
	var list []model.Exam
	for _, e := range m.examsByID {
		if e.UserID == userID {
			list = append(list, *e)
		}
	}
	return list, nil
}

func (m *MockExamRepository) Update(ctx context.Context, id, userID string, updates map[string]interface{}) (*model.Exam, error) {
	exam, ok := m.examsByID[id]
	if !ok || exam.UserID != userID {
		return nil, nil
	}
	if name, ok := updates["name"].(string); ok {
		exam.Name = name
	}
	return exam, nil
}

func (m *MockExamRepository) Delete(ctx context.Context, id, userID string) error {
	exam, ok := m.examsByID[id]
	if !ok || exam.UserID != userID {
		return apperror.NotFound("Exam not found")
	}
	delete(m.examsByID, id)
	return nil
}

func (m *MockExamRepository) ReplaceSyllabus(ctx context.Context, examID, userID string, subjects []model.Subject) (*model.Exam, error) {
	exam, ok := m.examsByID[examID]
	if !ok || exam.UserID != userID {
		return nil, nil
	}
	exam.Subjects = subjects
	return exam, nil
}

func TestCreateExamWithSyllabus(t *testing.T) {
	mockRepo := NewMockExamRepository()
	svc := service.NewExamService(mockRepo)

	examDate := time.Now().AddDate(0, 3, 0)
	hours := 4.0
	req := dto.CreateExamRequest{
		Name:                 "UPSC CSE 2026",
		ExamDate:             &examDate,
		AvailableHoursPerDay: &hours,
		Subjects: []dto.SubjectInput{
			{
				Name: "Polity",
				Topics: []dto.TopicInput{
					{
						Name: "Fundamental Rights",
						Subtopics: []dto.SubtopicInput{
							{Name: "Article 21"},
						},
					},
				},
			},
		},
	}

	res, err := svc.CreateExam(context.Background(), "user-123", req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if res.Name != "UPSC CSE 2026" {
		t.Errorf("expected name 'UPSC CSE 2026', got: %s", res.Name)
	}
	if len(res.Subjects) != 1 {
		t.Fatalf("expected 1 subject, got: %d", len(res.Subjects))
	}
	if res.Subjects[0].Name != "Polity" {
		t.Errorf("expected subject name 'Polity', got: %s", res.Subjects[0].Name)
	}
	if len(res.Subjects[0].Topics) != 1 {
		t.Fatalf("expected 1 topic, got: %d", len(res.Subjects[0].Topics))
	}
	if res.Subjects[0].Topics[0].Name != "Fundamental Rights" {
		t.Errorf("expected topic name 'Fundamental Rights', got: %s", res.Subjects[0].Topics[0].Name)
	}
}

func TestDuplicateSubjectRejection(t *testing.T) {
	mockRepo := NewMockExamRepository()
	svc := service.NewExamService(mockRepo)

	req := dto.CreateExamRequest{
		Name: "GATE",
		Subjects: []dto.SubjectInput{
			{Name: "Math"},
			{Name: "math"}, // Duplicate case-insensitive
		},
	}

	_, err := svc.CreateExam(context.Background(), "user-123", req)
	if err == nil {
		t.Fatal("expected duplicate subject error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.StatusCode != 400 {
		t.Errorf("expected 400 BadRequest AppError, got: %v", err)
	}
}

func TestDuplicateTopicUnderSameSubject(t *testing.T) {
	mockRepo := NewMockExamRepository()
	svc := service.NewExamService(mockRepo)

	req := dto.CreateExamRequest{
		Name: "GATE",
		Subjects: []dto.SubjectInput{
			{
				Name: "Math",
				Topics: []dto.TopicInput{
					{Name: "Calculus"},
					{Name: "calculus"},
				},
			},
		},
	}

	_, err := svc.CreateExam(context.Background(), "user-123", req)
	if err == nil {
		t.Fatal("expected duplicate topic error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.StatusCode != 400 {
		t.Errorf("expected 400 BadRequest AppError, got: %v", err)
	}
}

func TestGetExamNotFound(t *testing.T) {
	mockRepo := NewMockExamRepository()
	svc := service.NewExamService(mockRepo)

	_, err := svc.GetExamByID(context.Background(), "user-123", "non-existent-id")
	if err == nil {
		t.Fatal("expected 404 error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.StatusCode != 404 {
		t.Errorf("expected 404 NotFound AppError, got: %v", err)
	}
}

func TestExtractSyllabus_EmptyText_Returns400(t *testing.T) {
	mockRepo := NewMockExamRepository()
	svc := service.NewExamService(mockRepo)

	_, err := svc.ExtractSyllabus(context.Background(), "   ")
	if err == nil {
		t.Fatal("expected 400 error for empty text, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.StatusCode != 400 {
		t.Errorf("expected 400 BadRequest AppError, got: %v", err)
	}
}

func TestCreateExamRequest_FlexibleParsing(t *testing.T) {
	rawJSON := `{
		"name": "GATE CS 2027",
		"description": "Exam Description",
		"examDate": "2027-02-15",
		"availableHoursPerDay": "4.5",
		"subjects": [
			{
				"id": "sub-1",
				"name": "Databases",
				"topics": [
					{
						"id": "top-1",
						"name": "SQL",
						"subtopics": [
							{"id": "subtop-1", "name": "Joins"}
						]
					}
				]
			}
		]
	}`

	var req dto.CreateExamRequest
	if err := json.Unmarshal([]byte(rawJSON), &req); err != nil {
		t.Fatalf("failed to unmarshal flexible exam request: %v", err)
	}

	if req.ExamDate == nil || req.ExamDate.Year() != 2027 || req.ExamDate.Month() != 2 || req.ExamDate.Day() != 15 {
		t.Errorf("expected examDate 2027-02-15, got %v", req.ExamDate)
	}
	if req.AvailableHoursPerDay == nil || *req.AvailableHoursPerDay != 4.5 {
		t.Errorf("expected hours 4.5, got %v", req.AvailableHoursPerDay)
	}
	if len(req.Subjects) != 1 || req.Subjects[0].Name != "Databases" {
		t.Errorf("expected 1 subject 'Databases', got %v", req.Subjects)
	}
	if req.Subjects[0].Topics[0].Subtopics[0].Name != "Joins" {
		t.Errorf("expected subtopic 'Joins'")
	}

	// Test user's exact date string: "2027-02-24" and typo "2027-02--24"
	for _, dStr := range []string{"2027-02-24", "2027-02--24"} {
		typoJSON := `{"name":"Exam 1","examDate":"` + dStr + `"}`
		var typoReq dto.CreateExamRequest
		if err := json.Unmarshal([]byte(typoJSON), &typoReq); err != nil {
			t.Fatalf("failed to unmarshal with date %s: %v", dStr, err)
		}
		if typoReq.ExamDate == nil || typoReq.ExamDate.Day() != 24 {
			t.Errorf("expected day 24 for %s, got %v", dStr, typoReq.ExamDate)
		}
	}
}

func TestExtractTextFromFile_SanitizesInvalidUTF8(t *testing.T) {
	// Create a temporary file with invalid UTF-8 bytes and null bytes (\x00, 0xda, 0xed)
	tmpFile, err := os.CreateTemp("", "bad-utf8-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write invalid bytes like 0xda 0xed and null byte 0x00
	badContent := []byte{'H', 'e', 'l', 'l', 'o', 0xda, 0xed, 0x00, ' ', 'W', 'o', 'r', 'l', 'd'}
	if _, err := tmpFile.Write(badContent); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	cleanText, err := ai.ExtractTextFromFile(context.Background(), tmpFile.Name())
	if err != nil {
		t.Fatalf("ExtractTextFromFile failed: %v", err)
	}

	if cleanText != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", cleanText)
	}
}

func TestExtractTextFromFile_ImageFileWithoutClientReturnsEmpty(t *testing.T) {
	// Creating a fake image file with binary bytes
	tmpImg, err := os.CreateTemp("", "fake-notes-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpImg.Name())

	_, _ = tmpImg.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) // PNG header
	tmpImg.Close()

	// Calling without client should safely return empty string without error
	text, err := ai.ExtractTextFromFile(context.Background(), tmpImg.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty text for image without client, got %q", text)
	}
}

