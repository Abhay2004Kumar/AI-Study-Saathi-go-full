// Package service implements Resource Topic Mapping intelligence.
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/gemini"
	"github.com/google/uuid"
)

// TopicMappingService defines operations for mapping resources to syllabus topics.
type TopicMappingService interface {
	GetTopicResources(ctx context.Context, userID, examID, topicID string) (*dto.TopicResourcesResponse, error)
	GetDocumentMappings(ctx context.Context, userID, documentID string) ([]dto.EnrichedMappingItem, error)
	MapDocumentTopics(ctx context.Context, userID, documentID string) ([]dto.EnrichedMappingItem, error)
	UpdateMappingStatus(ctx context.Context, userID, mappingID, status string) (*model.ResourceTopicMapping, error)
	Run(ctx context.Context, documentID string) error
}

type topicMappingService struct {
	mappingRepo  repository.TopicMappingRepository
	examRepo     repository.ExamRepository
	docRepo      repository.DocumentRepository
	chunkRepo    repository.DocumentChunkRepository
	geminiClient *gemini.Client
}

// NewTopicMappingService creates a new TopicMappingService.
func NewTopicMappingService(
	mappingRepo repository.TopicMappingRepository,
	examRepo repository.ExamRepository,
	docRepo repository.DocumentRepository,
	chunkRepo repository.DocumentChunkRepository,
	geminiClient *gemini.Client,
) TopicMappingService {
	return &topicMappingService{
		mappingRepo:  mappingRepo,
		examRepo:     examRepo,
		docRepo:      docRepo,
		chunkRepo:    chunkRepo,
		geminiClient: geminiClient,
	}
}

type catalogRef struct {
	TopicID    string
	SubtopicID *string
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func buildCatalog(subjects []model.Subject) (string, map[string]catalogRef) {
	var lines []string
	targetByName := make(map[string]catalogRef)

	for _, sub := range subjects {
		for _, top := range sub.Topics {
			lines = append(lines, fmt.Sprintf("Topic: %q (in Subject %q)", top.Name, sub.Name))
			targetByName[normalizeName(top.Name)] = catalogRef{TopicID: top.ID, SubtopicID: nil}
			for _, st := range top.Subtopics {
				subID := st.ID
				lines = append(lines, fmt.Sprintf("Subtopic: %q (under Topic %q)", st.Name, top.Name))
				targetByName[normalizeName(st.Name)] = catalogRef{TopicID: top.ID, SubtopicID: &subID}
			}
		}
	}

	return strings.Join(lines, "\n"), targetByName
}

func (s *topicMappingService) GetTopicResources(ctx context.Context, userID, examID, topicID string) (*dto.TopicResourcesResponse, error) {
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

	rawMappings, err := s.mappingRepo.FindByTopicWithRelations(ctx, topicID, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching topic mappings: %w", err)
	}

	notes := make([]dto.TopicResourceMappingItem, 0)
	pyqs := make([]dto.TopicResourceMappingItem, 0)
	syllabus := make([]dto.TopicResourceMappingItem, 0)

	for _, m := range rawMappings {
		var subRef *dto.TopicResourceSubtopic
		if m.SubtopicID != nil && m.SubtopicName != nil {
			subRef = &dto.TopicResourceSubtopic{
				ID:   *m.SubtopicID,
				Name: *m.SubtopicName,
			}
		}

		item := dto.TopicResourceMappingItem{
			ID:         m.ID,
			StartPage:  m.StartPage,
			EndPage:    m.EndPage,
			Confidence: m.Confidence,
			Status:     m.Status,
			Subtopic:   subRef,
			Document: dto.TopicResourceDocument{
				ID:               m.DocID,
				Title:            m.DocTitle,
				Category:         m.DocCategory,
				FileName:         m.DocFileName,
				ProcessingStatus: m.DocProcessingStatus,
			},
		}

		cat := strings.ToUpper(m.DocCategory)
		switch {
		case cat == "NOTES" || cat == "GENERAL":
			notes = append(notes, item)
		case cat == "PYQ":
			pyqs = append(pyqs, item)
		case cat == "SYLLABUS":
			syllabus = append(syllabus, item)
		default:
			notes = append(notes, item)
		}
	}

	return &dto.TopicResourcesResponse{
		Topic: dto.TopicReference{
			ID:   targetTopic.ID,
			Name: targetTopic.Name,
		},
		Notes:    notes,
		Pyqs:     pyqs,
		Syllabus: syllabus,
	}, nil
}

func (s *topicMappingService) GetDocumentMappings(ctx context.Context, userID, documentID string) ([]dto.EnrichedMappingItem, error) {
	doc, err := s.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("fetching document: %w", err)
	}
	if doc == nil || doc.UserID != userID {
		return nil, apperror.NotFound("Document not found")
	}

	items, err := s.mappingRepo.FindByDocument(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("querying document mappings: %w", err)
	}

	result := make([]dto.EnrichedMappingItem, 0, len(items))
	for _, it := range items {
		var subRef *dto.TopicResourceSubtopic
		if it.SubtopicID != nil && it.SubtopicName != nil {
			subRef = &dto.TopicResourceSubtopic{
				ID:   *it.SubtopicID,
				Name: *it.SubtopicName,
			}
		}

		result = append(result, dto.EnrichedMappingItem{
			ID:         it.ID,
			DocumentID: it.DocumentID,
			TopicID:    it.TopicID,
			SubtopicID: it.SubtopicID,
			StartPage:  it.StartPage,
			EndPage:    it.EndPage,
			Confidence: it.Confidence,
			Status:     it.Status,
			CreatedAt:  it.CreatedAt,
			Topic: &dto.TopicReference{
				ID:   it.TopicID,
				Name: it.TopicName,
			},
			Subtopic: subRef,
		})
	}

	return result, nil
}

func (s *topicMappingService) MapDocumentTopics(ctx context.Context, userID, documentID string) ([]dto.EnrichedMappingItem, error) {
	doc, err := s.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("fetching document: %w", err)
	}
	if doc == nil || doc.UserID != userID {
		return nil, apperror.NotFound("Document not found")
	}
	if doc.ExamID == nil || *doc.ExamID == "" {
		return nil, apperror.BadRequest("Link this document to an exam before mapping it to topics")
	}

	if err := s.Run(ctx, documentID); err != nil {
		return nil, err
	}

	return s.GetDocumentMappings(ctx, userID, documentID)
}

func (s *topicMappingService) Run(ctx context.Context, documentID string) error {
	doc, err := s.docRepo.FindByID(ctx, documentID)
	if err != nil || doc == nil || doc.ExamID == nil || *doc.ExamID == "" {
		return nil // nothing to map
	}

	exam, err := s.examRepo.FindByID(ctx, *doc.ExamID, doc.UserID)
	if err != nil || exam == nil || len(exam.Subjects) == 0 {
		return nil // no syllabus to map against
	}

	catalogText, targetByName := buildCatalog(exam.Subjects)
	if len(targetByName) == 0 {
		return nil
	}

	chunks, err := s.chunkRepo.FindByDocumentID(ctx, documentID)
	if err != nil || len(chunks) == 0 {
		return nil // no chunks
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

	systemPrompt := `You are mapping a student's study resource onto their course syllabus.`
	userPrompt := fmt.Sprintf(`SYLLABUS TOPICS (choose ONLY from this list, using the exact name given):
%s

RESOURCE CONTENT:
%s

INSTRUCTIONS:
- For each syllabus topic or subtopic actually covered in the content, report the startPage and endPage (if page numbers are labeled, otherwise null) and your confidence (0.0 to 1.0).
- Use the exact name as given in the syllabus list — do not paraphrase, translate, or invent a new name.
- Skip any topic not genuinely covered in this content.
- Output raw JSON with structure: {"mappings": [{"topicName": "...", "startPage": 1, "endPage": 2, "confidence": 0.9}]}`, catalogText, content)

	var aiResult struct {
		Mappings []struct {
			TopicName  string   `json:"topicName"`
			StartPage  *int     `json:"startPage"`
			EndPage    *int     `json:"endPage"`
			Confidence *float64 `json:"confidence"`
		} `json:"mappings"`
	}

	if err := s.geminiClient.GenerateJSON(ctx, userPrompt, systemPrompt, 0, &aiResult); err != nil {
		return fmt.Errorf("gemini topic mapping error: %w", err)
	}

	var validMappings []model.ResourceTopicMapping
	for _, m := range aiResult.Mappings {
		target, ok := targetByName[normalizeName(m.TopicName)]
		if !ok {
			continue // AI hallucination or not in syllabus catalog
		}

		confidence := 0.0
		if m.Confidence != nil {
			confidence = *m.Confidence
		}

		status := "CONFIRMED"
		if confidence < 0.6 {
			status = "PENDING_REVIEW"
		}

		validMappings = append(validMappings, model.ResourceTopicMapping{
			ID:         uuid.New().String(),
			DocumentID: documentID,
			TopicID:    target.TopicID,
			SubtopicID: target.SubtopicID,
			StartPage:  m.StartPage,
			EndPage:    m.EndPage,
			Confidence: &confidence,
			Status:     status,
		})
	}

	return s.mappingRepo.ReplaceForDocument(ctx, documentID, validMappings)
}

func (s *topicMappingService) UpdateMappingStatus(ctx context.Context, userID, mappingID, status string) (*model.ResourceTopicMapping, error) {
	if status != "CONFIRMED" && status != "REJECTED" && status != "PENDING_REVIEW" {
		return nil, apperror.BadRequest("Status must be one of: CONFIRMED, REJECTED, PENDING_REVIEW")
	}

	mapping, err := s.mappingRepo.FindByID(ctx, mappingID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying mapping: %w", err)
	}
	if mapping == nil {
		return nil, apperror.NotFound("Mapping not found")
	}

	updated, err := s.mappingRepo.UpdateStatus(ctx, mappingID, status)
	if err != nil {
		return nil, fmt.Errorf("updating mapping status: %w", err)
	}

	return updated, nil
}

