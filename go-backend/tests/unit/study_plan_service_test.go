package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/google/uuid"
)

// MockStudyPlanRepository implements repository.StudyPlanRepository in memory
type MockStudyPlanRepository struct {
	plansByExamID map[string]*model.StudyPlan
	sessionsByID  map[string]*model.StudyPlanSession
}

func NewMockStudyPlanRepository() *MockStudyPlanRepository {
	return &MockStudyPlanRepository{
		plansByExamID: make(map[string]*model.StudyPlan),
		sessionsByID:  make(map[string]*model.StudyPlanSession),
	}
}

func (m *MockStudyPlanRepository) Upsert(ctx context.Context, plan *model.StudyPlan, sessions []model.StudyPlanSession) (*model.StudyPlan, error) {
	plan.ID = uuid.New().String()
	plan.GeneratedAt = time.Now().UTC()
	for i := range sessions {
		sessions[i].ID = uuid.New().String()
		sessions[i].PlanID = plan.ID
		m.sessionsByID[sessions[i].ID] = &sessions[i]
	}
	plan.Sessions = sessions
	m.plansByExamID[plan.ExamID] = plan
	return plan, nil
}

func (m *MockStudyPlanRepository) FindByExamAndUser(ctx context.Context, examID, userID string) (*model.StudyPlan, error) {
	plan, ok := m.plansByExamID[examID]
	if !ok || plan.UserID != userID {
		return nil, nil
	}
	return plan, nil
}

func (m *MockStudyPlanRepository) UpdateSessionCompletion(ctx context.Context, sessionID, userID string, completed bool) (*model.StudyPlanSession, error) {
	s, ok := m.sessionsByID[sessionID]
	if !ok {
		return nil, nil
	}
	s.Completed = completed
	if completed {
		now := time.Now().UTC()
		s.CompletedAt = &now
	} else {
		s.CompletedAt = nil
	}
	return s, nil
}

func TestStudyPlanService_Generate_Success(t *testing.T) {
	examDate := time.Now().UTC().AddDate(0, 0, 5) // 5 days from now
	hours := 2.0

	examRepo := NewMockExamRepository()
	examRepo.examsByID["exam-1"] = &model.Exam{
		ID:                   "exam-1",
		UserID:               "user-1",
		ExamDate:             &examDate,
		AvailableHoursPerDay: &hours,
		Subjects: []model.Subject{
			{
				ID: "sub-1",
				Topics: []model.Topic{
					{ID: "top-1", Name: "Binary Trees"},
					{ID: "top-2", Name: "Graph Traversals"},
				},
			},
		},
	}

	planRepo := NewMockStudyPlanRepository()
	masteryRepo := NewMockMasteryRepository()
	pyqRepo := NewMockPyqRepository()

	svc := service.NewStudyPlanService(planRepo, examRepo, masteryRepo, pyqRepo)
	resp, err := svc.Generate(context.Background(), "user-1", "exam-1", nil)
	if err != nil {
		t.Fatalf("unexpected generation error: %v", err)
	}

	if resp.Plan == nil {
		t.Fatalf("expected non-nil plan")
	}
	plan := resp.Plan.(*model.StudyPlan)
	if len(plan.Sessions) == 0 {
		t.Errorf("expected generated study sessions")
	}
}

func TestStudyPlanService_Generate_ValidationErrors(t *testing.T) {
	examRepo := NewMockExamRepository()
	planRepo := NewMockStudyPlanRepository()
	masteryRepo := NewMockMasteryRepository()
	pyqRepo := NewMockPyqRepository()
	svc := service.NewStudyPlanService(planRepo, examRepo, masteryRepo, pyqRepo)

	// 1. Missing Exam Date
	hours := 2.0
	examRepo.examsByID["exam-no-date"] = &model.Exam{
		ID:                   "exam-no-date",
		UserID:               "user-1",
		AvailableHoursPerDay: &hours,
	}
	_, err := svc.Generate(context.Background(), "user-1", "exam-no-date", nil)
	if err == nil || err.Error() != "Set an exam date before generating a study plan" {
		t.Errorf("expected missing exam date error, got: %v", err)
	}

	// 2. Missing Hours Per Day
	future := time.Now().UTC().AddDate(0, 0, 10)
	examRepo.examsByID["exam-no-hours"] = &model.Exam{
		ID:       "exam-no-hours",
		UserID:   "user-1",
		ExamDate: &future,
	}
	_, err = svc.Generate(context.Background(), "user-1", "exam-no-hours", nil)
	if err == nil || err.Error() != "Set your available study hours per day before generating a study plan" {
		t.Errorf("expected missing hours per day error, got: %v", err)
	}

	// 3. Past Exam Date
	past := time.Now().UTC().AddDate(0, 0, -2)
	examRepo.examsByID["exam-past"] = &model.Exam{
		ID:                   "exam-past",
		UserID:               "user-1",
		ExamDate:             &past,
		AvailableHoursPerDay: &hours,
	}
	_, err = svc.Generate(context.Background(), "user-1", "exam-past", nil)
	if err == nil || err.Error() != "This exam date has already passed" {
		t.Errorf("expected exam date passed error, got: %v", err)
	}
}

func TestStudyPlanService_SetSessionCompletion(t *testing.T) {
	planRepo := NewMockStudyPlanRepository()
	sessID := "sess-1"
	planRepo.sessionsByID[sessID] = &model.StudyPlanSession{
		ID:        sessID,
		Completed: false,
	}

	svc := service.NewStudyPlanService(planRepo, nil, nil, nil)
	updated, err := svc.SetSessionCompletion(context.Background(), "user-1", sessID, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !updated.Completed {
		t.Errorf("expected session to be marked completed")
	}
	if updated.CompletedAt == nil {
		t.Errorf("expected completedAt timestamp to be set")
	}
}
