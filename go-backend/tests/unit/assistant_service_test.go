package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/internal/service/ai"
	"github.com/aiStudySaathi/backend/pkg/gemini"
	"github.com/google/uuid"
)

// MockAssistantSessionRepository implements repository.AssistantSessionRepository in memory
type MockAssistantSessionRepository struct {
	sessionsByExamUser map[string]*model.AssistantSession
}

func NewMockAssistantSessionRepository() *MockAssistantSessionRepository {
	return &MockAssistantSessionRepository{
		sessionsByExamUser: make(map[string]*model.AssistantSession),
	}
}

func (m *MockAssistantSessionRepository) FindByExamAndUser(ctx context.Context, examID, userID string) (*model.AssistantSession, error) {
	key := examID + ":" + userID
	s, ok := m.sessionsByExamUser[key]
	if !ok {
		return nil, nil
	}
	return s, nil
}

func (m *MockAssistantSessionRepository) Upsert(ctx context.Context, examID, userID string, messages []byte) (*model.AssistantSession, error) {
	key := examID + ":" + userID
	s, ok := m.sessionsByExamUser[key]
	if !ok {
		s = &model.AssistantSession{
			ID:        uuid.New().String(),
			UserID:    userID,
			ExamID:    examID,
			CreatedAt: time.Now().UTC(),
		}
		m.sessionsByExamUser[key] = s
	}
	s.Messages = messages
	s.UpdatedAt = time.Now().UTC()
	return s, nil
}

func TestAssistantService_SendMessage_WithStudyPlanTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := gemini.GenerateContentResponse{
			Candidates: []gemini.Candidate{
				{
					Content: gemini.Content{
						Parts: []gemini.Part{
							{Text: "Today you have a 60-minute session scheduled on Virtual Memory."},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	geminiClient := gemini.NewClient("dummy-key", gemini.WithBaseURL(server.URL))
	assistantRepo := NewMockAssistantSessionRepository()
	examRepo := NewMockExamRepository()
	examRepo.examsByID["exam-1"] = &model.Exam{
		ID:     "exam-1",
		UserID: "user-1",
	}

	planRepo := NewMockStudyPlanRepository()
	// Populate active plan with a session today
	planRepo.plansByExamID["exam-1"] = &model.StudyPlan{
		ID:        "plan-1",
		ExamID:    "exam-1",
		UserID:    "user-1",
		StartDate: time.Now().UTC(),
		EndDate:   time.Now().UTC().AddDate(0, 0, 10),
		Sessions: []model.StudyPlanSession{
			{
				ID:              "sess-today",
				Date:            time.Now().UTC(),
				TopicID:         "top-vm",
				DurationMinutes: 60,
				Reason:          "Weak area",
			},
		},
	}
	planSvc := service.NewStudyPlanService(planRepo, examRepo, NewMockMasteryRepository(), NewMockPyqRepository())
	masterySvc := service.NewMasteryService(NewMockMasteryRepository(), examRepo)
	cardSvc := service.NewFlashcardService(NewMockFlashcardRepository(), examRepo, &MockChunkRepository{}, geminiClient)
	ragSvc := ai.NewRAGService(&MockChunkRepository{}, geminiClient)
	pyqRepo := NewMockPyqRepository()

	svc := ai.NewAssistantService(assistantRepo, examRepo, pyqRepo, ragSvc, masterySvc, planSvc, cardSvc, geminiClient)

	// User asks what to study today
	resp, err := svc.SendMessage(context.Background(), "user-1", "exam-1", "What is on my study plan for today?")
	if err != nil {
		t.Fatalf("unexpected assistant message error: %v", err)
	}

	if resp.Reply == "" {
		t.Errorf("expected non-empty assistant reply")
	}

	// Verify that get_todays_study_plan was executed
	foundTool := false
	for _, tool := range resp.ToolsUsed {
		if tool == "get_todays_study_plan" {
			foundTool = true
			break
		}
	}
	if !foundTool {
		t.Errorf("expected 'get_todays_study_plan' to be in toolsUsed, got %v", resp.ToolsUsed)
	}
}

