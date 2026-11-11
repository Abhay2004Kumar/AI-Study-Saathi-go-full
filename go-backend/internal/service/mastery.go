// Package service implements mastery computation logic.
package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/internal/service/scoring"
	"github.com/aiStudySaathi/backend/pkg/apperror"
)

// MasteryService defines operations for computing student mastery.
type MasteryService interface {
	GetTopicMastery(ctx context.Context, userID, examID, topicID string) (*dto.TopicMasteryResponse, error)
	GetExamMastery(ctx context.Context, userID, examID string) ([]dto.ExamMasteryItem, error)
}

type masteryService struct {
	masteryRepo repository.MasteryRepository
	examRepo    repository.ExamRepository
}

// NewMasteryService creates a new MasteryService.
func NewMasteryService(masteryRepo repository.MasteryRepository, examRepo repository.ExamRepository) MasteryService {
	return &masteryService{
		masteryRepo: masteryRepo,
		examRepo:    examRepo,
	}
}

func (s *masteryService) GetTopicMastery(ctx context.Context, userID, examID, topicID string) (*dto.TopicMasteryResponse, error) {
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

	answers, err := s.masteryRepo.FindAnswersByTopic(ctx, userID, topicID)
	if err != nil {
		return nil, fmt.Errorf("fetching mastery answers: %w", err)
	}

	mastery := scoring.ComputeTopicMastery(answers, time.Now().UTC(), nil)

	return &dto.TopicMasteryResponse{
		Topic: dto.TopicReference{
			ID:   targetTopic.ID,
			Name: targetTopic.Name,
		},
		Mastery: mastery,
	}, nil
}

func (s *masteryService) GetExamMastery(ctx context.Context, userID, examID string) ([]dto.ExamMasteryItem, error) {
	exam, err := s.examRepo.FindByID(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying exam: %w", err)
	}
	if exam == nil {
		return nil, apperror.NotFound("Exam not found")
	}

	var allTopics []model.Topic
	for _, sub := range exam.Subjects {
		allTopics = append(allTopics, sub.Topics...)
	}

	if len(allTopics) == 0 {
		return []dto.ExamMasteryItem{}, nil
	}

	byTopic, err := s.masteryRepo.FindAnswersByExam(ctx, userID, examID)
	if err != nil {
		return nil, fmt.Errorf("fetching exam mastery answers: %w", err)
	}

	now := time.Now().UTC()
	results := make([]dto.ExamMasteryItem, len(allTopics))
	for i, t := range allTopics {
		ans := byTopic[t.ID]
		results[i] = dto.ExamMasteryItem{
			Topic: dto.TopicReference{
				ID:   t.ID,
				Name: t.Name,
			},
			Mastery: scoring.ComputeTopicMastery(ans, now, nil),
		}
	}

	// Sort weakest-first (ascending by mastery.score) so student immediately sees what needs work
	sort.Slice(results, func(i, j int) bool {
		return results[i].Mastery.Score < results[j].Mastery.Score
	})

	return results, nil
}

