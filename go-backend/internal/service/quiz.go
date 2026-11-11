// Package service implements the business logic for AI Quiz Generation and Grading.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/gemini"
)

// QuizService defines operations for generating, taking, and grading quizzes.
type QuizService interface {
	GenerateTopicQuiz(ctx context.Context, userID, examID, topicID string, req dto.GenerateQuizRequest) (*dto.TakeableQuizResponse, error)
	ListTopicQuizzes(ctx context.Context, userID, examID, topicID string) ([]dto.TopicQuizListItemResponse, error)
	GetTakeableQuiz(ctx context.Context, userID, quizID string) (*dto.TakeableQuizResponse, error)
	SubmitAttempt(ctx context.Context, userID, quizID string, req dto.SubmitQuizRequest) (*dto.QuizAttemptDetailResponse, error)
	ListQuizAttempts(ctx context.Context, userID, quizID string) ([]dto.QuizAttemptListItem, error)
	GetQuizAttempt(ctx context.Context, userID, quizID, attemptID string) (*dto.QuizAttemptDetailResponse, error)
}

type quizService struct {
	quizRepo     repository.QuizRepository
	examRepo     repository.ExamRepository
	chunkRepo    repository.DocumentChunkRepository
	geminiClient *gemini.Client
}

// NewQuizService creates a new QuizService.
func NewQuizService(
	quizRepo repository.QuizRepository,
	examRepo repository.ExamRepository,
	chunkRepo repository.DocumentChunkRepository,
	geminiClient *gemini.Client,
) QuizService {
	return &quizService{
		quizRepo:     quizRepo,
		examRepo:     examRepo,
		chunkRepo:    chunkRepo,
		geminiClient: geminiClient,
	}
}

// LLMQuizQuestion matches the JSON schema expected from Gemini
type LLMQuizQuestion struct {
	Question      string   `json:"question"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correctAnswer"`
	Explanation   string   `json:"explanation"`
}

type LLMQuizOutput struct {
	Questions []LLMQuizQuestion `json:"questions"`
}

func (s *quizService) GenerateTopicQuiz(
	ctx context.Context,
	userID, examID, topicID string,
	req dto.GenerateQuizRequest,
) (*dto.TakeableQuizResponse, error) {
	// 1. Validate exam ownership and find topic
	exam, err := s.examRepo.FindByID(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying exam: %w", err)
	}
	if exam == nil {
		return nil, apperror.NotFound("Exam not found")
	}

	var targetTopic *model.Topic
	var targetSubject *model.Subject
	var subtopicName *string

	for _, sub := range exam.Subjects {
		for _, top := range sub.Topics {
			if top.ID == topicID {
				targetTopic = &top
				targetSubject = &sub
				if req.SubtopicID != "" {
					for _, st := range top.Subtopics {
						if st.ID == req.SubtopicID {
							subtopicName = &st.Name
							break
						}
					}
					if subtopicName == nil {
						return nil, apperror.NotFound("Subtopic not found under this topic")
					}
				}
				break
			}
		}
		if targetTopic != nil {
			break
		}
	}

	if targetTopic == nil {
		return nil, apperror.NotFound("Topic not found in this exam")
	}

	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "medium"
	}
	numQuestions := req.NumberOfQuestions
	if numQuestions <= 0 || numQuestions > 15 {
		numQuestions = 5
	}

	// 2. Retrieve study materials from pgvector for this topic
	searchQuery := targetTopic.Name
	if subtopicName != nil {
		searchQuery = fmt.Sprintf("%s %s", targetTopic.Name, *subtopicName)
	}

	queryVector, err := s.geminiClient.EmbedContent(ctx, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("generating query embedding: %w", err)
	}

	chunks, err := s.chunkRepo.SimilaritySearch(ctx, repository.SimilaritySearchParams{
		UserID:      userID,
		QueryVector: queryVector,
		TopK:        12,
		ExamID:      &examID,
	})
	if err != nil {
		return nil, fmt.Errorf("retrieving chunks for quiz: %w", err)
	}

	if len(chunks) == 0 {
		return nil, apperror.BadRequest("Not enough mapped material for this topic to generate a quiz yet — upload or map some notes first")
	}

	var contextBuilder strings.Builder
	docIDsSet := make(map[string]bool)
	for i, ch := range chunks {
		contextBuilder.WriteString(fmt.Sprintf("Source S%d (%s):\n%s\n\n", i+1, ch.Title, ch.Content))
		docIDsSet[ch.DocumentID] = true
	}

	docIDsList := make([]string, 0, len(docIDsSet))
	for id := range docIDsSet {
		docIDsList = append(docIDsList, id)
	}
	docIDsJSON, _ := json.Marshal(docIDsList)

	topicTitle := targetTopic.Name
	if subtopicName != nil {
		topicTitle = fmt.Sprintf("%s — %s", targetTopic.Name, *subtopicName)
	}

	// 3. Prompt Gemini with structured output
	prompt := fmt.Sprintf(`
You are an expert exam-preparation AI. Generate a multiple-choice quiz strictly from the provided study material.

TOPIC: %s
DIFFICULTY: %s
NUMBER OF QUESTIONS: %d
RANDOMIZATION SEED: %d

STUDY MATERIAL:
%s

INSTRUCTIONS:
- Generate exactly %d multiple-choice questions.
- Base every question ONLY on the study material above — never invent facts outside it.
- The difficulty of the questions should be '%s'.
- Each question must have exactly 4 options.
- Ensure correctAnswer exactly matches one of the options (character-for-character).
- Plain text only — no LaTeX and no markdown.
`, topicTitle, difficulty, numQuestions, rand.Int63(), contextBuilder.String(), numQuestions, difficulty)

	var llmOutput LLMQuizOutput
	err = s.geminiClient.GenerateJSON(ctx, prompt, "You generate rigorous, valid multiple-choice questions from provided texts.", 0.8, &llmOutput)
	if err != nil {
		return nil, fmt.Errorf("gemini quiz generation failed: %w", err)
	}

	// 4. Validate Questions (must have 4 options, correctAnswer must be in options)
	var validQuestions []model.QuizQuestion
	for _, q := range llmOutput.Questions {
		if len(q.Options) != 4 || q.Question == "" || q.CorrectAnswer == "" {
			continue
		}
		found := false
		for _, opt := range q.Options {
			if opt == q.CorrectAnswer {
				found = true
				break
			}
		}
		if !found {
			continue
		}

		optionsJSON, _ := json.Marshal(q.Options)
		explanation := q.Explanation
		validQuestions = append(validQuestions, model.QuizQuestion{
			Question:      q.Question,
			Options:       optionsJSON,
			CorrectAnswer: q.CorrectAnswer,
			Explanation:   &explanation,
			Difficulty:    difficulty,
		})
	}

	if len(validQuestions) == 0 {
		return nil, apperror.Internal("Failed to generate valid questions from study materials")
	}

	// 5. Persist Quiz and Questions
	var subtopicIDPtr *string
	if req.SubtopicID != "" {
		subtopicIDPtr = &req.SubtopicID
	}

	quiz := &model.Quiz{
		UserID:            userID,
		SubjectID:         targetSubject.ID,
		TopicID:           &targetTopic.ID,
		SubtopicID:        subtopicIDPtr,
		Title:             topicTitle,
		Difficulty:        difficulty,
		SourceDocumentIDs: docIDsJSON,
	}

	created, err := s.quizRepo.Create(ctx, quiz, validQuestions)
	if err != nil {
		return nil, fmt.Errorf("saving quiz: %w", err)
	}

	return s.toTakeableQuizResponse(created), nil
}

func (s *quizService) ListTopicQuizzes(ctx context.Context, userID, examID, topicID string) ([]dto.TopicQuizListItemResponse, error) {
	quizzes, err := s.quizRepo.FindByTopic(ctx, topicID, userID)
	if err != nil {
		return nil, fmt.Errorf("listing topic quizzes: %w", err)
	}

	result := make([]dto.TopicQuizListItemResponse, len(quizzes))
	for i, q := range quizzes {
		var lastAtt *dto.LastAttemptSummary
		if len(q.Attempts) > 0 {
			lastAtt = &dto.LastAttemptSummary{
				ID:          q.Attempts[0].ID,
				Score:       q.Attempts[0].Score,
				CompletedAt: q.Attempts[0].CompletedAt,
			}
		}

		result[i] = dto.TopicQuizListItemResponse{
			ID:            q.ID,
			Title:         q.Title,
			Difficulty:    q.Difficulty,
			SubtopicID:    q.SubtopicID,
			QuestionCount: len(q.Questions),
			CreatedAt:     q.CreatedAt,
			LastAttempt:   lastAtt,
		}
	}

	return result, nil
}

func (s *quizService) GetTakeableQuiz(ctx context.Context, userID, quizID string) (*dto.TakeableQuizResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, quizID, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching quiz: %w", err)
	}
	if quiz == nil {
		return nil, apperror.NotFound("Quiz not found")
	}

	return s.toTakeableQuizResponse(quiz), nil
}

func (s *quizService) SubmitAttempt(
	ctx context.Context,
	userID, quizID string,
	req dto.SubmitQuizRequest,
) (*dto.QuizAttemptDetailResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, quizID, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching quiz for grading: %w", err)
	}
	if quiz == nil {
		return nil, apperror.NotFound("Quiz not found")
	}

	// 1. Build lookup map of student answers: questionID -> selectedAnswer
	studentAnswers := make(map[string]string, len(req.Answers))
	for _, a := range req.Answers {
		if a.SelectedAnswer != nil {
			studentAnswers[a.QuestionID] = *a.SelectedAnswer
		}
	}

	// 2. Deterministic Server-Side Grading
	var correctCount int
	attemptAnswers := make([]model.QuizAttemptAnswer, len(quiz.Questions))
	gradedDetails := make([]dto.GradedAnswerDetail, len(quiz.Questions))

	for i, q := range quiz.Questions {
		selected, hasAnswer := studentAnswers[q.ID]
		var selectedPtr *string
		if hasAnswer {
			selectedPtr = &selected
		}

		isCorrect := hasAnswer && selected == q.CorrectAnswer
		if isCorrect {
			correctCount++
		}

		attemptAnswers[i] = model.QuizAttemptAnswer{
			QuestionID:     q.ID,
			SelectedAnswer: selectedPtr,
			IsCorrect:      isCorrect,
		}

		var opts []string
		_ = json.Unmarshal(q.Options, &opts)

		gradedDetails[i] = dto.GradedAnswerDetail{
			QuestionID:     q.ID,
			Question:       q.Question,
			Options:        opts,
			SelectedAnswer: selectedPtr,
			CorrectAnswer:  q.CorrectAnswer,
			Explanation:    q.Explanation,
			IsCorrect:      isCorrect,
		}
	}

	totalQuestions := len(quiz.Questions)
	score := 0.0
	if totalQuestions > 0 {
		score = math.Round((float64(correctCount)/float64(totalQuestions))*10000) / 100
	}

	attempt := &model.QuizAttempt{
		QuizID:         quiz.ID,
		UserID:         userID,
		Score:          score,
		TotalQuestions: totalQuestions,
		CorrectCount:   correctCount,
	}

	createdAttempt, err := s.quizRepo.CreateAttempt(ctx, attempt, attemptAnswers)
	if err != nil {
		return nil, fmt.Errorf("saving quiz attempt: %w", err)
	}

	return &dto.QuizAttemptDetailResponse{
		ID:             createdAttempt.ID,
		QuizID:         quiz.ID,
		Score:          createdAttempt.Score,
		CorrectCount:   createdAttempt.CorrectCount,
		TotalQuestions: createdAttempt.TotalQuestions,
		CompletedAt:    createdAttempt.CompletedAt,
		Answers:        gradedDetails,
	}, nil
}

func (s *quizService) ListQuizAttempts(ctx context.Context, userID, quizID string) ([]dto.QuizAttemptListItem, error) {
	attempts, err := s.quizRepo.FindAttemptsByQuiz(ctx, quizID, userID)
	if err != nil {
		return nil, fmt.Errorf("listing quiz attempts: %w", err)
	}

	result := make([]dto.QuizAttemptListItem, len(attempts))
	for i, a := range attempts {
		result[i] = dto.QuizAttemptListItem{
			ID:             a.ID,
			Score:          a.Score,
			CorrectCount:   a.CorrectCount,
			TotalQuestions: a.TotalQuestions,
			StartedAt:      a.StartedAt,
			CompletedAt:    a.CompletedAt,
		}
	}
	return result, nil
}

func (s *quizService) GetQuizAttempt(ctx context.Context, userID, quizID, attemptID string) (*dto.QuizAttemptDetailResponse, error) {
	attempt, err := s.quizRepo.FindAttemptByID(ctx, attemptID, quizID, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching attempt: %w", err)
	}
	if attempt == nil {
		return nil, apperror.NotFound("Attempt not found")
	}

	quiz, err := s.quizRepo.FindByID(ctx, quizID, userID)
	if err != nil {
		return nil, err
	}
	qMap := make(map[string]model.QuizQuestion)
	for _, q := range quiz.Questions {
		qMap[q.ID] = q
	}

	gradedDetails := make([]dto.GradedAnswerDetail, len(attempt.Answers))
	for i, a := range attempt.Answers {
		q := qMap[a.QuestionID]
		var opts []string
		_ = json.Unmarshal(q.Options, &opts)

		gradedDetails[i] = dto.GradedAnswerDetail{
			QuestionID:     q.ID,
			Question:       q.Question,
			Options:        opts,
			SelectedAnswer: a.SelectedAnswer,
			CorrectAnswer:  q.CorrectAnswer,
			Explanation:    q.Explanation,
			IsCorrect:      a.IsCorrect,
		}
	}

	return &dto.QuizAttemptDetailResponse{
		ID:             attempt.ID,
		QuizID:         quizID,
		Score:          attempt.Score,
		CorrectCount:   attempt.CorrectCount,
		TotalQuestions: attempt.TotalQuestions,
		CompletedAt:    attempt.CompletedAt,
		Answers:        gradedDetails,
	}, nil
}

func (s *quizService) toTakeableQuizResponse(quiz *model.Quiz) *dto.TakeableQuizResponse {
	questions := make([]dto.TakeableQuestion, len(quiz.Questions))
	for i, q := range quiz.Questions {
		var opts []string
		_ = json.Unmarshal(q.Options, &opts)
		questions[i] = dto.TakeableQuestion{
			ID:         q.ID,
			Question:   q.Question,
			Options:    opts,
			Difficulty: q.Difficulty,
		}
	}

	var sourceIDs interface{}
	if len(quiz.SourceDocumentIDs) > 0 {
		_ = json.Unmarshal(quiz.SourceDocumentIDs, &sourceIDs)
	}

	return &dto.TakeableQuizResponse{
		ID:                quiz.ID,
		Title:             quiz.Title,
		Difficulty:        quiz.Difficulty,
		TopicID:           quiz.TopicID,
		SubtopicID:        quiz.SubtopicID,
		SourceDocumentIDs: sourceIDs,
		CreatedAt:         quiz.CreatedAt,
		Questions:         questions,
	}
}

