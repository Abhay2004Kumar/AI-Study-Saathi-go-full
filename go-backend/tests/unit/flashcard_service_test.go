package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/pkg/gemini"
	"github.com/google/uuid"
)

// MockFlashcardRepository implements repository.FlashcardRepository in memory
type MockFlashcardRepository struct {
	cardsByID map[string]*model.Flashcard
}

func NewMockFlashcardRepository() *MockFlashcardRepository {
	return &MockFlashcardRepository{
		cardsByID: make(map[string]*model.Flashcard),
	}
}

func (m *MockFlashcardRepository) CreateMany(ctx context.Context, cards []model.Flashcard) error {
	for i := range cards {
		if cards[i].ID == "" {
			cards[i].ID = uuid.New().String()
		}
		m.cardsByID[cards[i].ID] = &cards[i]
	}
	return nil
}

func (m *MockFlashcardRepository) FindByTopic(ctx context.Context, topicID, userID string) ([]model.Flashcard, error) {
	var list []model.Flashcard
	for _, c := range m.cardsByID {
		if c.TopicID != nil && *c.TopicID == topicID && c.UserID == userID {
			list = append(list, *c)
		}
	}
	return list, nil
}

func (m *MockFlashcardRepository) FindDue(ctx context.Context, userID, examID string, limit int) ([]model.Flashcard, error) {
	var list []model.Flashcard
	now := time.Now().UTC()
	for _, c := range m.cardsByID {
		if c.UserID == userID && (c.DueAt.Before(now) || c.DueAt.Equal(now)) {
			list = append(list, *c)
			if len(list) >= limit {
				break
			}
		}
	}
	return list, nil
}

func (m *MockFlashcardRepository) FindByID(ctx context.Context, id, userID string) (*model.Flashcard, error) {
	c, ok := m.cardsByID[id]
	if !ok || c.UserID != userID {
		return nil, nil
	}
	return c, nil
}

func (m *MockFlashcardRepository) UpdateReview(ctx context.Context, id string, easeFactor float64, intervalDays, repetitions int, dueAt, lastReviewedAt time.Time) (*model.Flashcard, error) {
	c, ok := m.cardsByID[id]
	if !ok {
		return nil, nil
	}
	c.EaseFactor = easeFactor
	c.IntervalDays = intervalDays
	c.Repetitions = repetitions
	c.DueAt = dueAt
	c.LastReviewedAt = &lastReviewedAt
	return c, nil
}

func (m *MockFlashcardRepository) FindNormalizedFrontsByTopic(ctx context.Context, topicID, userID string) ([]string, error) {
	var fronts []string
	for _, c := range m.cardsByID {
		if c.TopicID != nil && *c.TopicID == topicID && c.UserID == userID {
			fronts = append(fronts, c.NormalizedFront)
		}
	}
	return fronts, nil
}

func TestFlashcardService_GenerateTopicFlashcards_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/models/gemini-embedding-2:embedContent" {
			_ = json.NewEncoder(w).Encode(gemini.EmbedContentResponse{
				Embedding: &gemini.ContentEmbedding{
					Values: []float32{0.05, 0.06},
				},
			})
			return
		}

		rawCardsJSON := `{
			"flashcards": [
				{
					"front": "What is thrashing in OS?",
					"back": "A state where the CPU spends more time paging than executing processes.",
					"hint": "Occurs when working sets exceed physical memory."
				}
			]
		}`
		resp := gemini.GenerateContentResponse{
			Candidates: []gemini.Candidate{
				{
					Content: gemini.Content{
						Parts: []gemini.Part{
							{Text: rawCardsJSON},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	geminiClient := gemini.NewClient("dummy-key", gemini.WithBaseURL(server.URL))
	cardRepo := NewMockFlashcardRepository()
	chunkRepo := &MockChunkRepository{
		chunks: []repository.SimilarityResult{
			{
				DocumentID: "doc-1",
				Title:      "Virtual Memory Notes",
				Content:    "Thrashing occurs when high paging activity degrades system throughput.",
			},
		},
	}
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

	svc := service.NewFlashcardService(cardRepo, examRepo, chunkRepo, geminiClient)
	created, err := svc.GenerateTopicFlashcards(context.Background(), "user-1", "exam-1", "top-1", dto.GenerateFlashcardsRequest{
		NumberOfCards: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(created) != 1 {
		t.Fatalf("expected 1 flashcard created, got %d", len(created))
	}
	if created[0].Front != "What is thrashing in OS?" {
		t.Errorf("unexpected front text: %s", created[0].Front)
	}
	if created[0].EaseFactor != 2.5 {
		t.Errorf("expected default ease factor 2.5, got %f", created[0].EaseFactor)
	}
}

func TestFlashcardService_ReviewFlashcard(t *testing.T) {
	cardRepo := NewMockFlashcardRepository()
	cardID := "card-test-1"
	cardRepo.cardsByID[cardID] = &model.Flashcard{
		ID:           cardID,
		UserID:       "user-1",
		Front:        "What is a Semaphore?",
		Back:         "A synchronization primitive.",
		EaseFactor:   2.5,
		IntervalDays: 0,
		Repetitions:  0,
		DueAt:        time.Now().UTC().Add(-1 * time.Hour), // Overdue
	}

	svc := service.NewFlashcardService(cardRepo, nil, nil, nil)

	// Review with GOOD
	updated, err := svc.ReviewFlashcard(context.Background(), "user-1", cardID, "GOOD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Repetitions != 1 {
		t.Errorf("expected repetitions=1, got %d", updated.Repetitions)
	}
	if updated.IntervalDays != 1 {
		t.Errorf("expected interval=1, got %d", updated.IntervalDays)
	}
	if updated.LastReviewedAt == nil {
		t.Errorf("expected lastReviewedAt to be populated")
	}
}

