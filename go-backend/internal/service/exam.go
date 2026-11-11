// Package service implements the core business logic for Exams and Syllabi.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/gemini"
)

// ExamService defines all operations for managing exams and their syllabus tree.
type ExamService interface {
	CreateExam(ctx context.Context, userID string, req dto.CreateExamRequest) (*dto.ExamDetailResponse, error)
	ListExams(ctx context.Context, userID string) ([]dto.ExamListItemResponse, error)
	GetExamByID(ctx context.Context, userID, examID string) (*dto.ExamDetailResponse, error)
	UpdateExam(ctx context.Context, userID, examID string, req dto.UpdateExamRequest) (*dto.ExamDetailResponse, error)
	DeleteExam(ctx context.Context, userID, examID string) error
	ReplaceSyllabus(ctx context.Context, userID, examID string, subjects []dto.SubjectInput) (*dto.ExamDetailResponse, error)
	ExtractSyllabus(ctx context.Context, rawText string) (*dto.ExtractSyllabusResponse, error)
}

type examService struct {
	examRepo     repository.ExamRepository
	geminiClient *gemini.Client
}

// NewExamService creates a new ExamService.
func NewExamService(examRepo repository.ExamRepository, geminiClient ...*gemini.Client) ExamService {
	var client *gemini.Client
	if len(geminiClient) > 0 {
		client = geminiClient[0]
	}
	return &examService{examRepo: examRepo, geminiClient: client}
}

// assertNoDuplicateNames checks that within the payload, no two items at the same level share a name.
// Matches Node.js exam.service.js: assertNoDuplicateNames()
func assertNoDuplicateNames(subjects []dto.SubjectInput) error {
	seenSubjects := make(map[string]bool)
	for _, sub := range subjects {
		key := strings.ToLower(strings.TrimSpace(sub.Name))
		if seenSubjects[key] {
			return apperror.BadRequest(fmt.Sprintf("Duplicate subject name %q", sub.Name))
		}
		seenSubjects[key] = true

		seenTopics := make(map[string]bool)
		for _, top := range sub.Topics {
			topKey := strings.ToLower(strings.TrimSpace(top.Name))
			if seenTopics[topKey] {
				return apperror.BadRequest(fmt.Sprintf("Duplicate topic name %q under subject %q", top.Name, sub.Name))
			}
			seenTopics[topKey] = true

			seenSubtopics := make(map[string]bool)
			for _, st := range top.Subtopics {
				stKey := strings.ToLower(strings.TrimSpace(st.Name))
				if seenSubtopics[stKey] {
					return apperror.BadRequest(fmt.Sprintf("Duplicate subtopic name %q under topic %q", st.Name, top.Name))
				}
				seenSubtopics[stKey] = true
			}
		}
	}
	return nil
}

// CreateExam creates an exam and optionally its syllabus tree.
func (s *examService) CreateExam(ctx context.Context, userID string, req dto.CreateExamRequest) (*dto.ExamDetailResponse, error) {
	if len(req.Subjects) > 0 {
		if err := assertNoDuplicateNames(req.Subjects); err != nil {
			return nil, err
		}
	}

	exam := &model.Exam{
		UserID:               userID,
		Name:                 req.Name,
		Description:          req.Description,
		ExamDate:             req.ExamDate,
		AvailableHoursPerDay: req.AvailableHoursPerDay,
	}

	created, err := s.examRepo.Create(ctx, exam, nil)
	if err != nil {
		return nil, fmt.Errorf("creating exam: %w", err)
	}

	if len(req.Subjects) > 0 {
		return s.ReplaceSyllabus(ctx, userID, created.ID, req.Subjects)
	}

	return s.GetExamByID(ctx, userID, created.ID)
}

// ListExams returns all exams for a user with subject counts.
func (s *examService) ListExams(ctx context.Context, userID string) ([]dto.ExamListItemResponse, error) {
	exams, err := s.examRepo.FindAllByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing exams: %w", err)
	}

	result := make([]dto.ExamListItemResponse, len(exams))
	for i, e := range exams {
		result[i] = dto.ExamListItemResponse{
			ID:                   e.ID,
			UserID:               e.UserID,
			Name:                 e.Name,
			Description:          e.Description,
			ExamDate:             e.ExamDate,
			AvailableHoursPerDay: e.AvailableHoursPerDay,
			Metadata:             e.Metadata,
			CreatedAt:            e.CreatedAt,
			UpdatedAt:            e.UpdatedAt,
			Count: dto.ExamCountInfo{
				Subjects: len(e.Subjects), // repository stored count here
			},
		}
	}

	return result, nil
}

// GetExamByID fetches a single exam with its entire syllabus tree.
func (s *examService) GetExamByID(ctx context.Context, userID, examID string) (*dto.ExamDetailResponse, error) {
	exam, err := s.examRepo.FindByID(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("getting exam: %w", err)
	}
	if exam == nil {
		return nil, apperror.NotFound("Exam not found")
	}

	return toExamDetailResponse(exam), nil
}

// UpdateExam updates specific fields on an exam.
func (s *examService) UpdateExam(ctx context.Context, userID, examID string, req dto.UpdateExamRequest) (*dto.ExamDetailResponse, error) {
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.ExamDate != nil {
		updates["examDate"] = req.ExamDate
	}
	if req.AvailableHoursPerDay != nil {
		updates["availableHoursPerDay"] = req.AvailableHoursPerDay
	}

	updated, err := s.examRepo.Update(ctx, examID, userID, updates)
	if err != nil {
		return nil, fmt.Errorf("updating exam: %w", err)
	}
	if updated == nil {
		return nil, apperror.NotFound("Exam not found")
	}

	return toExamDetailResponse(updated), nil
}

// DeleteExam removes an exam.
func (s *examService) DeleteExam(ctx context.Context, userID, examID string) error {
	err := s.examRepo.Delete(ctx, examID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return apperror.NotFound("Exam not found")
		}
		return fmt.Errorf("deleting exam: %w", err)
	}
	return nil
}

// ReplaceSyllabus reconciles the entire tree.
func (s *examService) ReplaceSyllabus(ctx context.Context, userID, examID string, subjectsInput []dto.SubjectInput) (*dto.ExamDetailResponse, error) {
	if err := assertNoDuplicateNames(subjectsInput); err != nil {
		return nil, err
	}

	// Verify exam exists and belongs to user
	existing, err := s.examRepo.FindByID(ctx, examID, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperror.NotFound("Exam not found")
	}

	// Map DTO inputs to Model structures
	models := make([]model.Subject, len(subjectsInput))
	for i, sub := range subjectsInput {
		topics := make([]model.Topic, len(sub.Topics))
		for j, top := range sub.Topics {
			subtopics := make([]model.Subtopic, len(top.Subtopics))
			for k, st := range top.Subtopics {
				subtopics[k] = model.Subtopic{
					Name:        st.Name,
					Description: st.Description,
					OrderIndex:  k,
				}
			}
			topics[j] = model.Topic{
				Name:        top.Name,
				Description: top.Description,
				OrderIndex:  j,
				Subtopics:   subtopics,
			}
		}
		models[i] = model.Subject{
			Name:        sub.Name,
			Description: sub.Description,
			OrderIndex:  i,
			Topics:      topics,
		}
	}

	reconciled, err := s.examRepo.ReplaceSyllabus(ctx, examID, userID, models)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "23505") {
			return nil, apperror.Conflict("Duplicate subject, topic, or subtopic name at the same level")
		}
		return nil, fmt.Errorf("replacing syllabus: %w", err)
	}

	return toExamDetailResponse(reconciled), nil
}

// Helper to convert Model to DTO detail response
func toExamDetailResponse(exam *model.Exam) *dto.ExamDetailResponse {
	subjects := make([]dto.SubjectResponse, len(exam.Subjects))
	for i, s := range exam.Subjects {
		topics := make([]dto.TopicResponse, len(s.Topics))
		for j, t := range s.Topics {
			subtopics := make([]dto.SubtopicResponse, len(t.Subtopics))
			for k, st := range t.Subtopics {
				subtopics[k] = dto.SubtopicResponse{
					ID:          st.ID,
					TopicID:     st.TopicID,
					Name:        st.Name,
					Description: st.Description,
					OrderIndex:  st.OrderIndex,
					CreatedAt:   st.CreatedAt,
					UpdatedAt:   st.UpdatedAt,
				}
			}
			topics[j] = dto.TopicResponse{
				ID:          t.ID,
				SubjectID:   t.SubjectID,
				Name:        t.Name,
				Description: t.Description,
				OrderIndex:  t.OrderIndex,
				CreatedAt:   t.CreatedAt,
				UpdatedAt:   t.UpdatedAt,
				Subtopics:   subtopics,
			}
		}
		subjects[i] = dto.SubjectResponse{
			ID:          s.ID,
			ExamID:      s.ExamID,
			UserID:      s.UserID,
			Name:        s.Name,
			Description: s.Description,
			OrderIndex:  s.OrderIndex,
			CreatedAt:   s.CreatedAt,
			UpdatedAt:   s.UpdatedAt,
			Topics:      topics,
		}
	}

	return &dto.ExamDetailResponse{
		ID:                   exam.ID,
		UserID:               exam.UserID,
		Name:                 exam.Name,
		Description:          exam.Description,
		ExamDate:             exam.ExamDate,
		AvailableHoursPerDay: exam.AvailableHoursPerDay,
		Metadata:             exam.Metadata,
		CreatedAt:            exam.CreatedAt,
		UpdatedAt:            exam.UpdatedAt,
		Subjects:             subjects,
	}
}

const maxSyllabusInputChars = 20000

// ExtractSyllabus analyzes raw syllabus text or document content and returns a suggested Subject -> Topic -> Subtopic tree.
// Matches Node.js SyllabusExtractionService.extractFromText()
func (s *examService) ExtractSyllabus(ctx context.Context, rawText string) (*dto.ExtractSyllabusResponse, error) {
	trimmed := strings.TrimSpace(rawText)
	if trimmed == "" {
		return nil, apperror.BadRequest("Provide syllabus text or upload a file to extract from")
	}

	if s.geminiClient == nil {
		return nil, apperror.Internal("AI service is not configured")
	}

	if len(trimmed) > maxSyllabusInputChars {
		trimmed = trimmed[:maxSyllabusInputChars]
	}

	prompt := fmt.Sprintf(`You are helping a student organize an exam syllabus into a structured outline.

RAW SYLLABUS TEXT:
%s

INSTRUCTIONS:
- Break the content into Subjects, each containing Topics, each optionally containing Subtopics.
- Preserve the original wording of names where reasonable; don't invent content the text doesn't imply.
- If the text is flat with no clear subject grouping, create a single subject capturing the overall exam/unit name and list topics under it.
- If a topic has no meaningfully distinct subtopics, return an empty subtopics array for it rather than inventing one.
- Return ONLY a JSON object matching this schema:
{
  "subjects": [
    {
      "name": "Subject Name",
      "topics": [
        {
          "name": "Topic Name",
          "subtopics": [
            { "name": "Subtopic Name" }
          ]
        }
      ]
    }
  ]
}
`, trimmed)

	var result dto.ExtractSyllabusResponse
	if err := s.geminiClient.GenerateJSON(ctx, prompt, "", 0.0, &result); err != nil {
		slog.Error("Failed to extract syllabus via Gemini", "error", err)
		return nil, apperror.Internal("Failed to extract a syllabus structure from the provided text")
	}

	if result.Subjects == nil {
		result.Subjects = []dto.ExtractedSubject{}
	}

	return &result, nil
}


