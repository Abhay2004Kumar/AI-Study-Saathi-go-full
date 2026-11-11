package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aiStudySaathi/backend/pkg/gemini"
)

func TestGeminiGenerateContent(t *testing.T) {
	// Spin up in-memory HTTP mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		resp := gemini.GenerateContentResponse{
			Candidates: []gemini.Candidate{
				{
					Content: gemini.Content{
						Parts: []gemini.Part{
							{Text: "Deadlocks can be prevented using Banker's Algorithm."},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := gemini.NewClient("dummy-key",
		gemini.WithBaseURL(server.URL),
	)

	text, err := client.GenerateContent(context.Background(), "Explain deadlocks", "", 0.2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Deadlocks can be prevented using Banker's Algorithm."
	if text != expected {
		t.Errorf("expected %q, got %q", expected, text)
	}
}

func TestGeminiGenerateJSON(t *testing.T) {
	type QuizQuestion struct {
		Question string   `json:"question"`
		Options  []string `json:"options"`
		Answer   string   `json:"answer"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawJSON := `{"question": "What is a semaphore?", "options": ["Mutex", "Counter", "Thread", "Lock"], "answer": "Counter"}`
		resp := gemini.GenerateContentResponse{
			Candidates: []gemini.Candidate{
				{
					Content: gemini.Content{
						Parts: []gemini.Part{
							{Text: "```json\n" + rawJSON + "\n```"},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := gemini.NewClient("dummy-key", gemini.WithBaseURL(server.URL))

	var question QuizQuestion
	err := client.GenerateJSON(context.Background(), "Generate question", "", 0.0, &question)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if question.Question != "What is a semaphore?" {
		t.Errorf("expected question 'What is a semaphore?', got: %s", question.Question)
	}
	if question.Answer != "Counter" {
		t.Errorf("expected answer 'Counter', got: %s", question.Answer)
	}
}

func TestGeminiEmbedContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := gemini.EmbedContentResponse{
			Embedding: &gemini.ContentEmbedding{
				Values: []float32{0.12, -0.45, 0.78},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := gemini.NewClient("dummy-key", gemini.WithBaseURL(server.URL))

	vec, err := client.EmbedContent(context.Background(), "test text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vec) != 3 {
		t.Fatalf("expected 3 dimensions, got: %d", len(vec))
	}
	if vec[0] != 0.12 {
		t.Errorf("expected vec[0] to be 0.12, got: %f", vec[0])
	}
}

func TestGeminiRetryOn429(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)
		if current == 1 {
			// First attempt returns 429 with retry hint
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": {"code": 429, "message": "Resource exhausted, retry in 0.1s"}}`))
			return
		}

		// Second attempt succeeds
		resp := gemini.GenerateContentResponse{
			Candidates: []gemini.Candidate{
				{
					Content: gemini.Content{
						Parts: []gemini.Part{
							{Text: "Recovered successfully after 429"},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := gemini.NewClient("dummy-key",
		gemini.WithBaseURL(server.URL),
		gemini.WithRetryConfig(gemini.RetryConfig{
			MaxRetries:     2,
			InitialBackoff: 50 * time.Millisecond,
			MaxBackoff:     200 * time.Millisecond,
		}),
	)

	text, err := client.GenerateContent(context.Background(), "Hello", "", 0.0)
	if err != nil {
		t.Fatalf("expected successful recovery on retry, got: %v", err)
	}

	if text != "Recovered successfully after 429" {
		t.Errorf("unexpected text: %s", text)
	}

	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected exactly 2 attempts, got: %d", attempts)
	}
}

