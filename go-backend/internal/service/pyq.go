// Package service implements Previous Year Question (PYQ) intelligence and scoring.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/internal/service/scoring"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/gemini"
	"github.com/google/uuid"
)

// PyqService defines operations for PYQ intelligence and status management.
type PyqService interface {
	GetTopicPyqAnalysis(ctx context.Context, userID, examID, topicID string) (*dto.TopicPyqAnalysisResponse, error)
	UpdateQuestionStatus(ctx context.Context, userID, questionID, status string) (*model.PyqQuestion, error)
	GetDocumentQuestions(ctx context.Context, userID, documentID string) ([]model.PyqQuestion, error)
	AnalyzePyqDocument(ctx context.Context, userID, documentID string) ([]model.PyqQuestion, error)
}

type pyqService struct {
	pyqRepo      repository.PyqRepository
	examRepo     repository.ExamRepository
	docRepo      repository.DocumentRepository
	chunkRepo    repository.DocumentChunkRepository
	geminiClient *gemini.Client
}

// NewPyqService creates a new PyqService.
func NewPyqService(
	pyqRepo repository.PyqRepository,
	examRepo repository.ExamRepository,
	docRepo repository.DocumentRepository,
	chunkRepo repository.DocumentChunkRepository,
	geminiClient *gemini.Client,
) PyqService {
	return &pyqService{
		pyqRepo:      pyqRepo,
		examRepo:     examRepo,
		docRepo:      docRepo,
		chunkRepo:    chunkRepo,
		geminiClient: geminiClient,
	}
}

func (s *pyqService) GetTopicPyqAnalysis(ctx context.Context, userID, examID, topicID string) (*dto.TopicPyqAnalysisResponse, error) {
	exam, err := s.examRepo.FindByID(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying exam: %w", err)
	}
	if exam == nil {
		return nil, apperror.NotFound("Exam not found")
	}

	var targetTopic *model.Topic
	for _, sub := range exam.Subjects {
		for _, top := range sub.Topics {
			if top.ID == topicID {
				targetTopic = &top
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

	pyqRows, err := s.pyqRepo.FindByTopic(ctx, topicID, userID, true)
	if err != nil {
		return nil, fmt.Errorf("fetching topic pyq questions: %w", err)
	}

	// Collect years for scoring
	var years []int
	for _, row := range pyqRows {
		if row.Question.Year != nil {
			years = append(years, *row.Question.Year)
		}
	}

	frequency := scoring.ComputeTopicFrequency(years, time.Now().Year(), nil)

	questionItems := make([]dto.TopicPyqQuestionItem, 0, len(pyqRows))
	for _, r := range pyqRows {
		var opts interface{}
		if r.Question.Options != nil {
			_ = json.Unmarshal(*r.Question.Options, &opts)
		}

		var subRef *dto.SubtopicReference
		if r.SubtopicID != nil && r.SubtopicName != nil {
			subRef = &dto.SubtopicReference{
				ID:   *r.SubtopicID,
				Name: *r.SubtopicName,
			}
		}

		docRef := &dto.DocumentReference{
			ID:    r.DocumentID,
			Title: r.DocumentTitle,
		}

		questionItems = append(questionItems, dto.TopicPyqQuestionItem{
			ID:            r.Question.ID,
			QuestionText:  r.Question.QuestionText,
			Options:       opts,
			CorrectAnswer: r.Question.CorrectAnswer,
			Year:          r.Question.Year,
			Confidence:    r.Question.Confidence,
			Status:        r.Question.Status,
			Subtopic:      subRef,
			Document:      docRef,
		})
	}

	return &dto.TopicPyqAnalysisResponse{
		Topic: dto.TopicReference{
			ID:   targetTopic.ID,
			Name: targetTopic.Name,
		},
		Frequency: frequency,
		Questions: questionItems,
	}, nil
}

func (s *pyqService) UpdateQuestionStatus(ctx context.Context, userID, questionID, status string) (*model.PyqQuestion, error) {
	if status != "CONFIRMED" && status != "REJECTED" && status != "PENDING_REVIEW" {
		return nil, apperror.BadRequest("Status must be one of: CONFIRMED, REJECTED, PENDING_REVIEW")
	}

	question, err := s.pyqRepo.FindByID(ctx, questionID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying question: %w", err)
	}
	if question == nil {
		return nil, apperror.NotFound("Question not found")
	}

	updated, err := s.pyqRepo.UpdateStatus(ctx, questionID, status)
	if err != nil {
		return nil, fmt.Errorf("updating question status: %w", err)
	}

	return updated, nil
}

func (s *pyqService) GetDocumentQuestions(ctx context.Context, userID, documentID string) ([]model.PyqQuestion, error) {
	doc, err := s.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("verifying document: %w", err)
	}
	if doc == nil || doc.UserID != userID {
		return nil, apperror.NotFound("Document not found")
	}

	questions, err := s.pyqRepo.FindByDocument(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("fetching questions for document: %w", err)
	}

	return questions, nil
}

func (s *pyqService) AnalyzePyqDocument(ctx context.Context, userID, documentID string) ([]model.PyqQuestion, error) {
	doc, err := s.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("verifying document: %w", err)
	}
	if doc == nil || doc.UserID != userID {
		return nil, apperror.NotFound("Document not found")
	}
	if doc.ExamID == nil || *doc.ExamID == "" {
		return nil, apperror.BadRequest("Link this document to an exam before analyzing its questions")
	}

	exam, err := s.examRepo.FindByID(ctx, *doc.ExamID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying exam: %w", err)
	}
	if exam == nil || len(exam.Subjects) == 0 {
		return nil, apperror.BadRequest("Nothing to analyze — this needs to be a PYQ resource with extracted questions and an exam that has a syllabus")
	}

	if s.chunkRepo == nil || s.geminiClient == nil {
		return nil, fmt.Errorf("AI dependencies not configured")
	}

	chunks, err := s.chunkRepo.FindByDocumentID(ctx, documentID)
	if err != nil || len(chunks) == 0 {
		return nil, apperror.BadRequest("Nothing to analyze — this needs to be a PYQ resource with extracted questions and an exam that has a syllabus")
	}

	catalogText, targetByName := buildCatalog(exam.Subjects)
	if len(targetByName) == 0 {
		return nil, apperror.BadRequest("Nothing to analyze — this needs to be a PYQ resource with extracted questions and an exam that has a syllabus")
	}

	var contentBuilder strings.Builder
	for _, chunk := range chunks {
		contentBuilder.WriteString(chunk.Content)
		contentBuilder.WriteString("\n\n")
		if contentBuilder.Len() > 20000 {
			break
		}
	}
	content := contentBuilder.String()
	if len(content) > 20000 {
		content = content[:20000]
	}

	systemPrompt := `You are extracting and mapping previous-year exam questions onto a course syllabus.`
	userPrompt := fmt.Sprintf(`SYLLABUS TOPICS (choose ONLY from this list, using the exact name given):
%s

CONTENT:
%s

INSTRUCTIONS:
- Extract each distinct exam question from the content.
- Include options array for multiple choice questions.
- Include correctAnswer if stated in the text.
- Include year as a 4-digit number (e.g. 2023) if stated in the text.
- For each question, map it to the SINGLE most relevant syllabus topic or subtopic from the syllabus list above.
- Return raw JSON with structure: {"questions": [{"questionText": "...", "options": ["A", "B"], "correctAnswer": "A", "year": 2023, "topicName": "...", "confidence": 0.85}]}`, catalogText, content)

	var aiResult struct {
		Questions []struct {
			QuestionText  string   `json:"questionText"`
			Options       []string `json:"options"`
			CorrectAnswer *string  `json:"correctAnswer"`
			Year          *int     `json:"year"`
			TopicName     string   `json:"topicName"`
			Confidence    *float64 `json:"confidence"`
		} `json:"questions"`
	}

	if err := s.geminiClient.GenerateJSON(ctx, userPrompt, systemPrompt, 0, &aiResult); err != nil {
		return nil, fmt.Errorf("gemini question mapping error: %w", err)
	}

	var questions []model.PyqQuestion
	for _, q := range aiResult.Questions {
		if strings.TrimSpace(q.QuestionText) == "" {
			continue
		}

		target, ok := targetByName[normalizeName(q.TopicName)]
		var topicID *string
		var subtopicID *string
		confidence := 0.0
		if q.Confidence != nil {
			confidence = *q.Confidence
		}
		status := "PENDING_REVIEW"
		if ok {
			topicID = &target.TopicID
			subtopicID = target.SubtopicID
			if confidence >= 0.6 {
				status = "CONFIRMED"
			}
		}

		var optionsBytes *[]byte
		if len(q.Options) > 0 {
			b, _ := json.Marshal(q.Options)
			optionsBytes = &b
		}

		questions = append(questions, model.PyqQuestion{
			ID:             uuid.New().String(),
			DocumentID:     documentID,
			TopicID:        topicID,
			SubtopicID:     subtopicID,
			QuestionText:   q.QuestionText,
			NormalizedText: normalizeName(q.QuestionText),
			Options:        optionsBytes,
			CorrectAnswer:  q.CorrectAnswer,
			Year:           q.Year,
			Confidence:     &confidence,
			Status:         status,
		})
	}

	if err := s.pyqRepo.ReplaceForDocument(ctx, documentID, questions); err != nil {
		return nil, fmt.Errorf("persisting pyq questions: %w", err)
	}

	return s.pyqRepo.FindByDocument(ctx, documentID)
}
