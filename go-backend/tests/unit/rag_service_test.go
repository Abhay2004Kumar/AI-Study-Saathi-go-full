package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/internal/service/ai"
	"github.com/aiStudySaathi/backend/pkg/gemini"
)

// MockChunkRepository is an in-memory mock for DocumentChunkRepository
type MockChunkRepository struct {
	chunks []repository.SimilarityResult
}

func (m *MockChunkRepository) InsertWithEmbedding(ctx context.Context, chunk *model.DocumentChunk, embedding []float32) error {
	return nil
}

func (m *MockChunkRepository) InsertWithoutEmbedding(ctx context.Context, chunk *model.DocumentChunk) error {
	return nil
}

func (m *MockChunkRepository) SimilaritySearch(ctx context.Context, params repository.SimilaritySearchParams) ([]repository.SimilarityResult, error) {
	return m.chunks, nil
}

func (m *MockChunkRepository) FindByDocumentID(ctx context.Context, documentID string) ([]model.DocumentChunk, error) {
	return nil, nil
}

func (m *MockChunkRepository) DeleteByDocumentID(ctx context.Context, documentID string) error {
	return nil
}

func TestRAGAskQuestionAndValidateSources(t *testing.T) {
	// Mock server that returns answers with one valid citation ("S1") and one hallucinated ("S99")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/models/gemini-embedding-2:embedContent" {
			_ = json.NewEncoder(w).Encode(gemini.EmbedContentResponse{
				Embedding: &gemini.ContentEmbedding{
					Values: []float32{0.1, 0.2, 0.3},
				},
			})
			return
		}

		rawJSON := `{"answer": "A semaphore is a synchronization primitive.", "foundInMaterials": true, "usedSourceIds": ["S1", "S99"]}`
		resp := gemini.GenerateContentResponse{
			Candidates: []gemini.Candidate{
				{
					Content: gemini.Content{
						Parts: []gemini.Part{
							{Text: rawJSON},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	geminiClient := gemini.NewClient("dummy-key", gemini.WithBaseURL(server.URL))

	mockChunkRepo := &MockChunkRepository{
		chunks: []repository.SimilarityResult{
			{
				Title:   "OS Chapter 5 Notes",
				Content: "A semaphore S is an integer variable accessed via wait() and signal().",
			},
		},
	}

	ragSvc := ai.NewRAGService(mockChunkRepo, geminiClient)

	resp, err := ragSvc.AskQuestion(context.Background(), "user-1", "What is a semaphore?", ai.RAGFilters{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.FoundInMaterials {
		t.Error("expected foundInMaterials to be true")
	}

	// Verify defensive source resolution:
	// Only S1 was valid ("OS Chapter 5 Notes").
	// S99 was hallucinated and MUST be stripped out!
	if len(resp.Sources) != 1 {
		t.Fatalf("expected exactly 1 verified source, got: %d", len(resp.Sources))
	}

	if resp.Sources[0].Title != "OS Chapter 5 Notes" {
		t.Errorf("expected source title 'OS Chapter 5 Notes', got: %s", resp.Sources[0].Title)
	}
}

