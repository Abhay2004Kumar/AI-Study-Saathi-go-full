package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/internal/service"
)

// MockMasteryRepository implements repository.MasteryRepository in memory
type MockMasteryRepository struct {
	answersByTopic map[string][]repository.MasteryAnswer
}

func NewMockMasteryRepository() *MockMasteryRepository {
	return &MockMasteryRepository{
		answersByTopic: make(map[string][]repository.MasteryAnswer),
	}
}

func (m *MockMasteryRepository) FindAnswersByTopic(ctx context.Context, userID, topicID string) ([]repository.MasteryAnswer, error) {
	return m.answersByTopic[topicID], nil
}

func (m *MockMasteryRepository) FindAnswersByExam(ctx context.Context, userID, examID string) (map[string][]repository.MasteryAnswer, error) {
	return m.answersByTopic, nil
}

// MockPyqRepository implements repository.PyqRepository in memory
type MockPyqRepository struct {
	questionsByTopic map[string][]repository.PyqWithRelations
	questionsByID    map[string]*model.PyqQuestion
}

func NewMockPyqRepository() *MockPyqRepository {
	return &MockPyqRepository{
		questionsByTopic: make(map[string][]repository.PyqWithRelations),
		questionsByID:    make(map[string]*model.PyqQuestion),
	}
}

func (m *MockPyqRepository) FindByTopic(ctx context.Context, topicID, userID string, excludeRejected bool) ([]repository.PyqWithRelations, error) {
	return m.questionsByTopic[topicID], nil
}

func (m *MockPyqRepository) FindByExam(ctx context.Context, examID string, excludeRejected bool) (map[string][]model.PyqQuestion, error) {
	return nil, nil
}

func (m *MockPyqRepository) FindByID(ctx context.Context, id, userID string) (*model.PyqQuestion, error) {
	return m.questionsByID[id], nil
}

func (m *MockPyqRepository) UpdateStatus(ctx context.Context, id, status string) (*model.PyqQuestion, error) {
	q, ok := m.questionsByID[id]
	if !ok {
		return nil, nil
	}
	q.Status = status
	return q, nil
}

func (m *MockPyqRepository) FindByDocument(ctx context.Context, documentID string) ([]model.PyqQuestion, error) {
	var list []model.PyqQuestion
	for _, q := range m.questionsByID {
		if q.DocumentID == documentID {
			list = append(list, *q)
		}
	}
	return list, nil
}

func (m *MockPyqRepository) ReplaceForDocument(ctx context.Context, documentID string, questions []model.PyqQuestion) error {
	return nil
}

func TestMasteryService_GetExamMastery_SortedWeakestFirst(t *testing.T) {
	examRepo := NewMockExamRepository()
	examRepo.examsByID["exam-1"] = &model.Exam{
		ID:     "exam-1",
		UserID: "user-1",
		Subjects: []model.Subject{
			{
				ID:     "sub-1",
				ExamID: "exam-1",
				Topics: []model.Topic{
					{ID: "top-strong", Name: "Easy Concept"},
					{ID: "top-weak", Name: "Hard Concept"},
				},
			},
		},
	}

	now := time.Now().UTC()
	masteryRepo := NewMockMasteryRepository()
	// Strong topic: all correct
	masteryRepo.answersByTopic["top-strong"] = []repository.MasteryAnswer{
		{IsCorrect: true, CompletedAt: now},
		{IsCorrect: true, CompletedAt: now},
		{IsCorrect: true, CompletedAt: now},
	}
	// Weak topic: all wrong
	masteryRepo.answersByTopic["top-weak"] = []repository.MasteryAnswer{
		{IsCorrect: false, CompletedAt: now},
		{IsCorrect: false, CompletedAt: now},
		{IsCorrect: false, CompletedAt: now},
	}

	svc := service.NewMasteryService(masteryRepo, examRepo)
	results, err := svc.GetExamMastery(context.Background(), "user-1", "exam-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(results))
	}

	// Verify weakest-first sorting: top-weak (score 0.0) must come before top-strong (score 1.0)
	if results[0].Topic.ID != "top-weak" {
		t.Errorf("expected first topic to be weakest ('top-weak'), got %s", results[0].Topic.ID)
	}
	if results[0].Mastery.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", results[0].Mastery.Score)
	}
	if results[1].Topic.ID != "top-strong" {
		t.Errorf("expected second topic to be strongest ('top-strong'), got %s", results[1].Topic.ID)
	}
	if results[1].Mastery.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f", results[1].Mastery.Score)
	}
}

func TestPyqService_GetTopicPyqAnalysis(t *testing.T) {
	examRepo := NewMockExamRepository()
	examRepo.examsByID["exam-1"] = &model.Exam{
		ID:     "exam-1",
		UserID: "user-1",
		Subjects: []model.Subject{
			{
				ID: "sub-1",
				Topics: []model.Topic{
					{ID: "top-1", Name: "Virtual Memory"},
				},
			},
		},
	}

	pyqRepo := NewMockPyqRepository()
	year2024 := 2024
	year2025 := 2025
	pyqRepo.questionsByTopic["top-1"] = []repository.PyqWithRelations{
		{
			Question: model.PyqQuestion{
				ID:           "q-1",
				QuestionText: "Explain page faults in demand paging.",
				Year:         &year2024,
				Status:       "CONFIRMED",
			},
			DocumentID:    "doc-1",
			DocumentTitle: "Gate 2024 CS Paper",
		},
		{
			Question: model.PyqQuestion{
				ID:           "q-2",
				QuestionText: "Calculate effective memory access time with TLB.",
				Year:         &year2025,
				Status:       "CONFIRMED",
			},
			DocumentID:    "doc-2",
			DocumentTitle: "Gate 2025 CS Paper",
		},
	}

	svc := service.NewPyqService(pyqRepo, examRepo, nil, nil, nil)
	resp, err := svc.GetTopicPyqAnalysis(context.Background(), "user-1", "exam-1", "top-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Topic.Name != "Virtual Memory" {
		t.Errorf("expected topic name 'Virtual Memory', got %s", resp.Topic.Name)
	}
	if len(resp.Questions) != 2 {
		t.Errorf("expected 2 questions, got %d", len(resp.Questions))
	}
	if resp.Frequency.QuestionCount != 2 {
		t.Errorf("expected question count 2, got %d", resp.Frequency.QuestionCount)
	}
	if len(resp.Frequency.YearsSeen) != 2 {
		t.Errorf("expected 2 years seen (2024, 2025), got %d", len(resp.Frequency.YearsSeen))
	}
}

func TestPyqService_UpdateQuestionStatus(t *testing.T) {
	pyqRepo := NewMockPyqRepository()
	pyqRepo.questionsByID["q-100"] = &model.PyqQuestion{
		ID:     "q-100",
		Status: "PENDING_REVIEW",
	}

	svc := service.NewPyqService(pyqRepo, nil, nil, nil, nil)

	// Invalid status rejection
	_, err := svc.UpdateQuestionStatus(context.Background(), "user-1", "q-100", "INVALID_STATUS")
	if err == nil {
		t.Fatalf("expected error for invalid status")
	}

	// Valid status update
	updated, err := svc.UpdateQuestionStatus(context.Background(), "user-1", "q-100", "CONFIRMED")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != "CONFIRMED" {
		t.Errorf("expected status CONFIRMED, got %s", updated.Status)
	}
}

