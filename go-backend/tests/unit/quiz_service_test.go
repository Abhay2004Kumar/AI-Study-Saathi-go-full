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

// MockQuizRepository implements repository.QuizRepository in memory
type MockQuizRepository struct {
	quizzes  map[string]*model.Quiz
	attempts map[string]*model.QuizAttempt
}

func NewMockQuizRepository() *MockQuizRepository {
	return &MockQuizRepository{
		quizzes:  make(map[string]*model.Quiz),
		attempts: make(map[string]*model.QuizAttempt),
	}
}

func (m *MockQuizRepository) Create(ctx context.Context, quiz *model.Quiz, questions []model.QuizQuestion) (*model.Quiz, error) {
	quiz.ID = uuid.New().String()
	for i := range questions {
		questions[i].ID = uuid.New().String()
		questions[i].QuizID = quiz.ID
	}
	quiz.Questions = questions
	m.quizzes[quiz.ID] = quiz
	return quiz, nil
}

func (m *MockQuizRepository) FindByID(ctx context.Context, id, userID string) (*model.Quiz, error) {
	q, ok := m.quizzes[id]
	if !ok || q.UserID != userID {
		return nil, nil
	}
	return q, nil
}

func (m *MockQuizRepository) FindByTopic(ctx context.Context, topicID, userID string) ([]model.Quiz, error) {
	var result []model.Quiz
	for _, q := range m.quizzes {
		if q.TopicID != nil && *q.TopicID == topicID && q.UserID == userID {
			result = append(result, *q)
		}
	}
	return result, nil
}

func (m *MockQuizRepository) CreateAttempt(ctx context.Context, attempt *model.QuizAttempt, answers []model.QuizAttemptAnswer) (*model.QuizAttempt, error) {
	attempt.ID = uuid.New().String()
	now := time.Now().UTC()
	attempt.CompletedAt = &now
	for i := range answers {
		answers[i].ID = uuid.New().String()
		answers[i].AttemptID = attempt.ID
	}
	attempt.Answers = answers
	m.attempts[attempt.ID] = attempt
	return attempt, nil
}

func (m *MockQuizRepository) FindAttemptByID(ctx context.Context, attemptID, quizID, userID string) (*model.QuizAttempt, error) {
	a, ok := m.attempts[attemptID]
	if !ok || a.QuizID != quizID || a.UserID != userID {
		return nil, nil
	}
	return a, nil
}

func (m *MockQuizRepository) FindAttemptsByQuiz(ctx context.Context, quizID, userID string) ([]model.QuizAttempt, error) {
	var result []model.QuizAttempt
	for _, a := range m.attempts {
		if a.QuizID == quizID && a.UserID == userID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func TestQuizService_GenerateTopicQuiz_Success(t *testing.T) {
	// 1. Mock Gemini server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/models/gemini-embedding-2:embedContent" {
			_ = json.NewEncoder(w).Encode(gemini.EmbedContentResponse{
				Embedding: &gemini.ContentEmbedding{
					Values: []float32{0.01, 0.02, 0.03},
				},
			})
			return
		}

		rawQuizJSON := `{
			"questions": [
				{
					"question": "What is the primary function of a mutex in concurrent programming?",
					"options": [
						"To allocate heap memory",
						"To enforce mutual exclusion on shared resources",
						"To schedule CPU processes",
						"To compress network packets"
					],
					"correctAnswer": "To enforce mutual exclusion on shared resources",
					"explanation": "A mutex prevents concurrent access to critical sections."
				}
			]
		}`
		resp := gemini.GenerateContentResponse{
			Candidates: []gemini.Candidate{
				{
					Content: gemini.Content{
						Parts: []gemini.Part{
							{Text: rawQuizJSON},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	geminiClient := gemini.NewClient("dummy-key", gemini.WithBaseURL(server.URL))
	quizRepo := NewMockQuizRepository()
	chunkRepo := &MockChunkRepository{
		chunks: []repository.SimilarityResult{
			{
				DocumentID: "doc-1",
				Title:      "Concurrency in Go",
				Content:    "A sync.Mutex guarantees mutual exclusion across goroutines.",
			},
		},
	}
	examRepo := NewMockExamRepository()
	examRepo.examsByID["exam-1"] = &model.Exam{
		ID:     "exam-1",
		UserID: "user-1",
		Subjects: []model.Subject{
			{
				ID:     "sub-1",
				ExamID: "exam-1",
				Topics: []model.Topic{
					{
						ID:        "top-1",
						SubjectID: "sub-1",
						Name:      "Concurrency Primitives",
					},
				},
			},
		},
	}

	svc := service.NewQuizService(quizRepo, examRepo, chunkRepo, geminiClient)

	resp, err := svc.GenerateTopicQuiz(context.Background(), "user-1", "exam-1", "top-1", dto.GenerateQuizRequest{
		Difficulty:        "medium",
		NumberOfQuestions: 1,
	})
	if err != nil {
		t.Fatalf("expected successful quiz generation, got: %v", err)
	}

	if resp.ID == "" {
		t.Errorf("expected non-empty quiz ID")
	}
	if len(resp.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(resp.Questions))
	}
	if len(resp.Questions[0].Options) != 4 {
		t.Errorf("expected 4 options, got %d", len(resp.Questions[0].Options))
	}
	// Verify that takeable response does NOT leak the answer
	if resp.Questions[0].Question != "What is the primary function of a mutex in concurrent programming?" {
		t.Errorf("unexpected question text: %s", resp.Questions[0].Question)
	}
}

func TestQuizService_GenerateTopicQuiz_InsufficientMaterials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(gemini.EmbedContentResponse{
			Embedding: &gemini.ContentEmbedding{
				Values: []float32{0.1, 0.2},
			},
		})
	}))
	defer server.Close()

	geminiClient := gemini.NewClient("dummy-key", gemini.WithBaseURL(server.URL))
	quizRepo := NewMockQuizRepository()
	chunkRepo := &MockChunkRepository{chunks: []repository.SimilarityResult{}} // Empty!
	examRepo := NewMockExamRepository()
	examRepo.examsByID["exam-1"] = &model.Exam{
		ID:     "exam-1",
		UserID: "user-1",
		Subjects: []model.Subject{
			{
				ID: "sub-1",
				Topics: []model.Topic{
					{ID: "top-1", Name: "Empty Topic"},
				},
			},
		},
	}

	svc := service.NewQuizService(quizRepo, examRepo, chunkRepo, geminiClient)
	_, err := svc.GenerateTopicQuiz(context.Background(), "user-1", "exam-1", "top-1", dto.GenerateQuizRequest{})
	if err == nil {
		t.Fatalf("expected error when study material is missing, got nil")
	}
}

func TestQuizService_SubmitAttempt_ExactMatchGrading(t *testing.T) {
	quizRepo := NewMockQuizRepository()
	examRepo := NewMockExamRepository()
	chunkRepo := &MockChunkRepository{}
	geminiClient := gemini.NewClient("dummy-key")

	svc := service.NewQuizService(quizRepo, examRepo, chunkRepo, geminiClient)

	// Pre-populate a quiz directly in mock repo
	q1ID := "q-1"
	q2ID := "q-2"
	optsJSON, _ := json.Marshal([]string{"Option A", "Option B", "Option C", "Option D"})
	expl := "Explanation text"

	createdQuiz, _ := quizRepo.Create(context.Background(), &model.Quiz{
		UserID:     "user-1",
		SubjectID:  "sub-1",
		Title:      "Grading Test Quiz",
		Difficulty: "medium",
	}, []model.QuizQuestion{
		{
			ID:            q1ID,
			Question:      "Question 1?",
			Options:       optsJSON,
			CorrectAnswer: "Option B",
			Explanation:   &expl,
			Difficulty:    "medium",
		},
		{
			ID:            q2ID,
			Question:      "Question 2?",
			Options:       optsJSON,
			CorrectAnswer: "Option A",
			Explanation:   &expl,
			Difficulty:    "medium",
		},
	})

	// Student submits Q1 correct ("Option B") and Q2 incorrect ("Option C")
	ans1 := "Option B"
	ans2 := "Option C"
	submission := dto.SubmitQuizRequest{
		Answers: []dto.SubmitAnswerItem{
			{QuestionID: createdQuiz.Questions[0].ID, SelectedAnswer: &ans1},
			{QuestionID: createdQuiz.Questions[1].ID, SelectedAnswer: &ans2},
		},
	}

	attemptResp, err := svc.SubmitAttempt(context.Background(), "user-1", createdQuiz.ID, submission)
	if err != nil {
		t.Fatalf("unexpected submission error: %v", err)
	}

	if attemptResp.TotalQuestions != 2 {
		t.Errorf("expected total questions 2, got %d", attemptResp.TotalQuestions)
	}
	if attemptResp.CorrectCount != 1 {
		t.Errorf("expected 1 correct, got %d", attemptResp.CorrectCount)
	}
	if attemptResp.Score != 50.0 {
		t.Errorf("expected score 50.0, got %f", attemptResp.Score)
	}

	// Verify graded details
	if !attemptResp.Answers[0].IsCorrect {
		t.Errorf("expected answer 1 to be marked correct")
	}
	if attemptResp.Answers[1].IsCorrect {
		t.Errorf("expected answer 2 to be marked incorrect")
	}
	if attemptResp.Answers[0].CorrectAnswer != "Option B" {
		t.Errorf("expected answer 1 correct answer to be Option B, got %s", attemptResp.Answers[0].CorrectAnswer)
	}
}

func TestQuizService_GetTakeableQuiz_SanitizesAnswers(t *testing.T) {
	quizRepo := NewMockQuizRepository()
	optsJSON, _ := json.Marshal([]string{"X", "Y", "Z", "W"})
	expl := "Secret Explanation"

	createdQuiz, _ := quizRepo.Create(context.Background(), &model.Quiz{
		UserID:     "user-1",
		SubjectID:  "sub-1",
		Title:      "Sanitization Test",
		Difficulty: "hard",
	}, []model.QuizQuestion{
		{
			Question:      "What is secret?",
			Options:       optsJSON,
			CorrectAnswer: "Y",
			Explanation:   &expl,
			Difficulty:    "hard",
		},
	})

	svc := service.NewQuizService(quizRepo, NewMockExamRepository(), &MockChunkRepository{}, gemini.NewClient("dummy"))
	takeable, err := svc.GetTakeableQuiz(context.Background(), "user-1", createdQuiz.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(takeable.Questions) != 1 {
		t.Fatalf("expected 1 question")
	}
	// Ensure takeable question struct has NO correctAnswer or explanation field
	if takeable.Questions[0].Question != "What is secret?" {
		t.Errorf("unexpected question: %s", takeable.Questions[0].Question)
	}
	if len(takeable.Questions[0].Options) != 4 {
		t.Errorf("expected 4 options, got %d", len(takeable.Questions[0].Options))
	}
}
