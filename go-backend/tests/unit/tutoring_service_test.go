package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/service/ai"
	"github.com/aiStudySaathi/backend/pkg/gemini"
	"github.com/google/uuid"
)

// MockTutorSessionRepository implements repository.TutorSessionRepository in memory
type MockTutorSessionRepository struct {
	sessionsByID map[string]*model.TutorSession
}

func NewMockTutorSessionRepository() *MockTutorSessionRepository {
	return &MockTutorSessionRepository{
		sessionsByID: make(map[string]*model.TutorSession),
	}
}

func (m *MockTutorSessionRepository) Create(ctx context.Context, session *model.TutorSession) error {
	session.ID = uuid.New().String()
	m.sessionsByID[session.ID] = session
	return nil
}

func (m *MockTutorSessionRepository) FindByID(ctx context.Context, id, userID string) (*model.TutorSession, error) {
	s, ok := m.sessionsByID[id]
	if !ok || s.UserID != userID {
		return nil, nil
	}
	return s, nil
}

func (m *MockTutorSessionRepository) Update(ctx context.Context, id string, updates map[string]interface{}) (*model.TutorSession, error) {
	s, ok := m.sessionsByID[id]
	if !ok {
		return nil, nil
	}
	if status, ok := updates["status"].(string); ok {
		s.Status = status
	}
	if concept, ok := updates["currentConcept"].(*string); ok {
		s.CurrentConcept = concept
	}
	if hist, ok := updates["conversationHistory"].([]byte); ok {
		s.ConversationHistory = hist
	}
	if weak, ok := updates["weakAreas"].([]byte); ok {
		s.WeakAreas = weak
	}
	if count, ok := updates["questionCount"].(int); ok {
		s.QuestionCount = count
	}
	return s, nil
}

func TestTutoringService_StartSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/models/gemini-embedding-2:embedContent" {
			_ = json.NewEncoder(w).Encode(gemini.EmbedContentResponse{
				Embedding: &gemini.ContentEmbedding{Values: []float32{0.1, 0.2}},
			})
			return
		}

		resp := gemini.GenerateContentResponse{
			Candidates: []gemini.Candidate{
				{
					Content: gemini.Content{
						Parts: []gemini.Part{
							{Text: "EXPLANATION: Paging divides physical memory into fixed-size frames.\nQUESTION: What is the benefit of fixed-size frames?"},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	geminiClient := gemini.NewClient("dummy-key", gemini.WithBaseURL(server.URL))
	tutorRepo := NewMockTutorSessionRepository()
	chunkRepo := &MockChunkRepository{}

	svc := ai.NewTutoringService(tutorRepo, chunkRepo, geminiClient)
	resp, err := svc.StartSession(context.Background(), "user-1", "Operating Systems", "Paging")
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	if resp.SessionID == "" {
		t.Errorf("expected session ID")
	}
	if resp.SessionStatus != "ACTIVE" {
		t.Errorf("expected ACTIVE status, got %s", resp.SessionStatus)
	}
	if resp.QuestionCount != 1 {
		t.Errorf("expected questionCount 1, got %d", resp.QuestionCount)
	}
}

func TestTutoringService_Respond_WeakBranch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// First call is evaluate (returns weak JSON), second call is explainAgain
		evalJSON := `{"evaluation": "weak", "feedback": "Almost! But frames are physical, pages are logical.", "weakArea": "Page vs Frame distinction"}`
		resp := gemini.GenerateContentResponse{
			Candidates: []gemini.Candidate{
				{
					Content: gemini.Content{
						Parts: []gemini.Part{
							{Text: evalJSON},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	geminiClient := gemini.NewClient("dummy-key", gemini.WithBaseURL(server.URL))
	tutorRepo := NewMockTutorSessionRepository()
	sessionID := "sess-weak-test"
	currConcept := "Paging Basics"
	hist, _ := json.Marshal([]ai.HistoryMessage{{Role: "ai", Content: "Opening question"}})
	weak, _ := json.Marshal([]string{})

	tutorRepo.sessionsByID[sessionID] = &model.TutorSession{
		ID:                  sessionID,
		UserID:              "user-1",
		Subject:             "Operating Systems",
		Topic:               "Virtual Memory",
		Status:              "ACTIVE",
		CurrentConcept:      &currConcept,
		ConversationHistory: hist,
		WeakAreas:           weak,
		QuestionCount:       1,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}

	svc := ai.NewTutoringService(tutorRepo, nil, geminiClient)
	resp, err := svc.Respond(context.Background(), "user-1", sessionID, "A frame is inside a process.")
	if err != nil {
		t.Fatalf("unexpected respond error: %v", err)
	}

	if resp.QuestionCount != 2 {
		t.Errorf("expected question count 2, got %d", resp.QuestionCount)
	}
	if len(resp.WeakAreas) == 0 {
		t.Errorf("expected weak area to be recorded on weak evaluation")
	}
}

func TestTutoringService_Respond_MaxQuestionsCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		evalJSON := `{"evaluation": "strong", "feedback": "Perfect answer!", "weakArea": ""}`
		resp := gemini.GenerateContentResponse{
			Candidates: []gemini.Candidate{
				{
					Content: gemini.Content{
						Parts: []gemini.Part{
							{Text: evalJSON},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	geminiClient := gemini.NewClient("dummy-key", gemini.WithBaseURL(server.URL))
	tutorRepo := NewMockTutorSessionRepository()
	sessionID := "sess-complete-test"
	currConcept := "TLB Hit Ratio"
	hist, _ := json.Marshal([]ai.HistoryMessage{})
	weak, _ := json.Marshal([]string{"Earlier weak topic"})

	tutorRepo.sessionsByID[sessionID] = &model.TutorSession{
		ID:                  sessionID,
		UserID:              "user-1",
		Subject:             "Operating Systems",
		Topic:               "Memory Management",
		Status:              "ACTIVE",
		CurrentConcept:      &currConcept,
		ConversationHistory: hist,
		WeakAreas:           weak,
		QuestionCount:       5, // 5th question answered -> becomes 6th -> triggers completion
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}

	svc := ai.NewTutoringService(tutorRepo, nil, geminiClient)
	resp, err := svc.Respond(context.Background(), "user-1", sessionID, "Effective access time = hit_ratio * tlb_time + miss_ratio * mem_time.")
	if err != nil {
		t.Fatalf("unexpected respond error: %v", err)
	}

	if resp.SessionStatus != "COMPLETED" {
		t.Errorf("expected sessionStatus COMPLETED upon hitting question limit, got %s", resp.SessionStatus)
	}
	if resp.QuestionCount != 6 {
		t.Errorf("expected questionCount 6, got %d", resp.QuestionCount)
	}
}

