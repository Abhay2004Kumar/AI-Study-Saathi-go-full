// Package service implements flashcard generation, deduplication, and SM-2 spaced repetition.
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

// FlashcardService defines operations for flashcards and spaced repetition.
type FlashcardService interface {
	GenerateTopicFlashcards(ctx context.Context, userID, examID, topicID string, req dto.GenerateFlashcardsRequest) ([]model.Flashcard, error)
	ListTopicFlashcards(ctx context.Context, userID, examID, topicID string) ([]model.Flashcard, error)
	GetDueFlashcards(ctx context.Context, userID, examID string, limit int) ([]model.Flashcard, error)
	ReviewFlashcard(ctx context.Context, userID, cardID, grade string) (*model.Flashcard, error)
}

type flashcardService struct {
	cardRepo     repository.FlashcardRepository
	examRepo     repository.ExamRepository
	chunkRepo    repository.DocumentChunkRepository
	geminiClient *gemini.Client
}

// NewFlashcardService creates a new FlashcardService.
func NewFlashcardService(
	cardRepo repository.FlashcardRepository,
	examRepo repository.ExamRepository,
	chunkRepo repository.DocumentChunkRepository,
	geminiClient *gemini.Client,
) FlashcardService {
	return &flashcardService{
		cardRepo:     cardRepo,
		examRepo:     examRepo,
		chunkRepo:    chunkRepo,
		geminiClient: geminiClient,
	}
}

// LLMFlashcardOutput matches the JSON output from Gemini
type LLMFlashcardOutput struct {
	Flashcards []scoring.RawFlashcard `json:"flashcards"`
}

func (s *flashcardService) GenerateTopicFlashcards(
	ctx context.Context,
	userID, examID, topicID string,
	req dto.GenerateFlashcardsRequest,
) ([]model.Flashcard, error) {
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

	count := req.NumberOfCards
	if count <= 0 || count > 20 {
		count = 10
	}

	// 2. Retrieve study materials from pgvector
	searchQuery := fmt.Sprintf("%s key concepts and terms", targetTopic.Name)
	if subtopicName != nil {
		searchQuery = fmt.Sprintf("%s %s key concepts and terms", targetTopic.Name, *subtopicName)
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
		return nil, fmt.Errorf("retrieving chunks for flashcards: %w", err)
	}

	if len(chunks) == 0 {
		return nil, apperror.BadRequest("Not enough mapped material for this topic to generate flashcards yet — upload or map some notes first")
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
You are an expert study-aid AI specializing in concise, effective flashcards.
Generate a flashcard deck based ONLY on the provided study material.

TOPIC: %s
NUMBER OF CARDS: %d

STUDY MATERIAL:
%s

INSTRUCTIONS:
- Generate exactly %d flashcards.
- Base every card ONLY on the study material above — never invent facts outside it.
- Each card must cover a distinct fact or concept — no near-duplicates.
- "front" is a clear, concise question or term. "back" is the answer or definition, brief but complete.
- Optionally include a short "hint" that nudges without revealing the answer.
- Plain text only — no LaTeX and no markdown.
`, topicTitle, count, contextBuilder.String(), count)

	var llmOutput LLMFlashcardOutput
	err = s.geminiClient.GenerateJSON(ctx, prompt, "You generate concise, high-yield flashcards from study materials.", 0.3, &llmOutput)
	if err != nil {
		return nil, fmt.Errorf("gemini flashcard generation failed: %w", err)
	}

	// 4. Fetch existing normalized fronts for deduplication
	existingFronts, err := s.cardRepo.FindNormalizedFrontsByTopic(ctx, topicID, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching existing flashcards for deduplication: %w", err)
	}

	existingSet := make(map[string]bool, len(existingFronts))
	for _, f := range existingFronts {
		existingSet[f] = true
	}

	deduped := scoring.DedupeFlashcards(llmOutput.Flashcards, existingSet)
	if len(deduped) == 0 {
		return []model.Flashcard{}, nil
	}

	// 5. Prepare and persist new cards
	now := time.Now().UTC()
	var subtopicIDPtr *string
	if req.SubtopicID != "" {
		subtopicIDPtr = &req.SubtopicID
	}

	newCards := make([]model.Flashcard, len(deduped))
	for i, c := range deduped {
		var hintPtr *string
		if c.Hint != "" {
			hint := c.Hint
			hintPtr = &hint
		}

		newCards[i] = model.Flashcard{
			ID:                uuid.New().String(),
			UserID:            userID,
			SubjectID:         targetSubject.ID,
			TopicID:           &targetTopic.ID,
			SubtopicID:        subtopicIDPtr,
			Front:             c.Front,
			Back:              c.Back,
			Hint:              hintPtr,
			NormalizedFront:   scoring.NormalizeFrontText(c.Front),
			SourceDocumentIDs: docIDsJSON,
			EaseFactor:        scoring.DefaultEaseFactor,
			IntervalDays:      0,
			Repetitions:       0,
			DueAt:             now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
	}

	if err := s.cardRepo.CreateMany(ctx, newCards); err != nil {
		return nil, fmt.Errorf("persisting flashcards: %w", err)
	}

	return newCards, nil
}

func (s *flashcardService) ListTopicFlashcards(ctx context.Context, userID, examID, topicID string) ([]model.Flashcard, error) {
	exam, err := s.examRepo.FindByID(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying exam: %w", err)
	}
	if exam == nil {
		return nil, apperror.NotFound("Exam not found")
	}

	cards, err := s.cardRepo.FindByTopic(ctx, topicID, userID)
	if err != nil {
		return nil, fmt.Errorf("listing topic flashcards: %w", err)
	}
	return cards, nil
}

func (s *flashcardService) GetDueFlashcards(ctx context.Context, userID, examID string, limit int) ([]model.Flashcard, error) {
	exam, err := s.examRepo.FindByID(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying exam: %w", err)
	}
	if exam == nil {
		return nil, apperror.NotFound("Exam not found")
	}

	return s.cardRepo.FindDue(ctx, userID, examID, limit)
}

func (s *flashcardService) ReviewFlashcard(ctx context.Context, userID, cardID, grade string) (*model.Flashcard, error) {
	if _, ok := scoring.GradeQuality[grade]; !ok {
		return nil, apperror.BadRequest("grade must be one of: AGAIN, HARD, GOOD, EASY")
	}

	card, err := s.cardRepo.FindByID(ctx, cardID, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching flashcard: %w", err)
	}
	if card == nil {
		return nil, apperror.NotFound("Flashcard not found")
	}

	now := time.Now().UTC()
	state := scoring.SM2State{
		EaseFactor:   card.EaseFactor,
		IntervalDays: card.IntervalDays,
		Repetitions:  card.Repetitions,
	}

	sm2Res, err := scoring.ApplyReview(state, grade, now)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	updated, err := s.cardRepo.UpdateReview(
		ctx,
		card.ID,
		sm2Res.EaseFactor,
		sm2Res.IntervalDays,
		sm2Res.Repetitions,
		sm2Res.DueAt,
		sm2Res.LastReviewedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("saving flashcard review: %w", err)
	}

	return updated, nil
}

