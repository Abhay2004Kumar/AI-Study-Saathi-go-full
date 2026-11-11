// Package service implements the study planning orchestration service.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/internal/service/scoring"
	"github.com/aiStudySaathi/backend/pkg/apperror"
)

// StudyPlanService defines operations for generating and managing adaptive study plans.
type StudyPlanService interface {
	Generate(ctx context.Context, userID, examID string, cfg *scoring.StudyPlanningConfig) (*dto.GenerateStudyPlanResponse, error)
	GetActivePlan(ctx context.Context, userID, examID string) (*model.StudyPlan, error)
	SetSessionCompletion(ctx context.Context, userID, sessionID string, completed bool) (*model.StudyPlanSession, error)
}

type studyPlanService struct {
	planRepo    repository.StudyPlanRepository
	examRepo    repository.ExamRepository
	masteryRepo repository.MasteryRepository
	pyqRepo     repository.PyqRepository
}

// NewStudyPlanService creates a new StudyPlanService.
func NewStudyPlanService(
	planRepo repository.StudyPlanRepository,
	examRepo repository.ExamRepository,
	masteryRepo repository.MasteryRepository,
	pyqRepo repository.PyqRepository,
) StudyPlanService {
	return &studyPlanService{
		planRepo:    planRepo,
		examRepo:    examRepo,
		masteryRepo: masteryRepo,
		pyqRepo:     pyqRepo,
	}
}

func (s *studyPlanService) Generate(
	ctx context.Context,
	userID, examID string,
	cfg *scoring.StudyPlanningConfig,
) (*dto.GenerateStudyPlanResponse, error) {
	// 1. Verify exam ownership and requirements
	exam, err := s.examRepo.FindByID(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying exam: %w", err)
	}
	if exam == nil {
		return nil, apperror.NotFound("Exam not found")
	}
	if exam.ExamDate == nil {
		return nil, apperror.BadRequest("Set an exam date before generating a study plan")
	}
	if exam.AvailableHoursPerDay == nil || *exam.AvailableHoursPerDay <= 0 {
		return nil, apperror.BadRequest("Set your available study hours per day before generating a study plan")
	}

	today := time.Now().UTC()
	if scoring.StartOfDay(*exam.ExamDate).Before(scoring.StartOfDay(today)) {
		return nil, apperror.BadRequest("This exam date has already passed")
	}

	// 2. Collect syllabus topics
	var allTopics []model.Topic
	for _, sub := range exam.Subjects {
		allTopics = append(allTopics, sub.Topics...)
	}
	if len(allTopics) == 0 {
		return nil, apperror.BadRequest("Add a syllabus with topics before generating a study plan")
	}

	// 3. Batch query mastery answers and PYQ questions across the entire exam
	masteryByTopic, err := s.masteryRepo.FindAnswersByExam(ctx, userID, examID)
	if err != nil {
		return nil, fmt.Errorf("fetching exam mastery: %w", err)
	}

	pyqByTopic, err := s.pyqRepo.FindByExam(ctx, examID, true)
	if err != nil {
		return nil, fmt.Errorf("fetching exam pyq questions: %w", err)
	}

	// 4. Build input records for SWRR
	currentYear := today.Year()
	inputs := make([]scoring.SWRRTopicInput, len(allTopics))
	for i, t := range allTopics {
		masteryAns := masteryByTopic[t.ID]
		mastery := scoring.ComputeTopicMastery(masteryAns, today, nil)

		var importanceScore *float64
		if pyqs, ok := pyqByTopic[t.ID]; ok && len(pyqs) > 0 {
			var years []int
			for _, q := range pyqs {
				if q.Year != nil {
					years = append(years, *q.Year)
				}
			}
			freq := scoring.ComputeTopicFrequency(years, currentYear, nil)
			importanceScore = &freq.Score
		}

		inputs[i] = scoring.SWRRTopicInput{
			TopicID:         t.ID,
			TopicName:       t.Name,
			MasteryScore:    mastery.Score,
			MasteryLevel:    mastery.Level,
			LowConfidence:   mastery.LowConfidence,
			ImportanceScore: importanceScore,
		}
	}

	// 5. Run SWRR allocation algorithm
	built := scoring.BuildStudyPlan(inputs, today, *exam.ExamDate, *exam.AvailableHoursPerDay, cfg)

	// 6. Flatten daily sessions into database rows
	var sessionsToInsert []model.StudyPlanSession
	for _, day := range built.Days {
		for _, sess := range day.Sessions {
			sessionsToInsert = append(sessionsToInsert, model.StudyPlanSession{
				Date:            day.Date,
				TopicID:         sess.TopicID,
				SubtopicID:      sess.SubtopicID,
				DurationMinutes: sess.DurationMinutes,
				PriorityScore:   sess.PriorityScore,
				Reason:          sess.Reason,
				Completed:       false,
				Topic:           &model.Topic{ID: sess.TopicID, Name: sess.TopicName},
			})
		}
	}

	// 7. Persist atomic replacement
	planToCreate := &model.StudyPlan{
		ExamID:    examID,
		UserID:    userID,
		StartDate: scoring.StartOfDay(today),
		EndDate:   scoring.StartOfDay(*exam.ExamDate),
	}

	persistedPlan, err := s.planRepo.Upsert(ctx, planToCreate, sessionsToInsert)
	if err != nil {
		return nil, fmt.Errorf("saving study plan: %w", err)
	}

	return &dto.GenerateStudyPlanResponse{
		Plan:            persistedPlan,
		SkippedTopicIDs: built.SkippedTopicIDs,
	}, nil
}

func (s *studyPlanService) GetActivePlan(ctx context.Context, userID, examID string) (*model.StudyPlan, error) {
	exam, err := s.examRepo.FindByID(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying exam: %w", err)
	}
	if exam == nil {
		return nil, apperror.NotFound("Exam not found")
	}

	return s.planRepo.FindByExamAndUser(ctx, examID, userID)
}

func (s *studyPlanService) SetSessionCompletion(ctx context.Context, userID, sessionID string, completed bool) (*model.StudyPlanSession, error) {
	session, err := s.planRepo.UpdateSessionCompletion(ctx, sessionID, userID, completed)
	if err != nil {
		return nil, fmt.Errorf("updating session completion: %w", err)
	}
	if session == nil {
		return nil, apperror.NotFound("Study plan session not found")
	}
	return session, nil
}

