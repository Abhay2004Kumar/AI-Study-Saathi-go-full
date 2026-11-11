package service_test

import (
	"testing"
	"time"

	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/internal/service/scoring"
)

func TestComputeTopicMastery_Empty(t *testing.T) {
	now := time.Now().UTC()
	res := scoring.ComputeTopicMastery(nil, now, nil)

	if res.Score != 0 || res.Level != "NOT_STARTED" || !res.LowConfidence || res.AttemptCount != 0 {
		t.Errorf("unexpected empty mastery result: %+v", res)
	}
}

func TestComputeTopicMastery_RecencyDecay(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

	// Scenario: Student was brilliant 42 days ago (2 half-lives back), but answered wrong today.
	// 42 days ago: weight = 0.5^(42/21) = 0.25
	// Today (0 days ago): weight = 0.5^0 = 1.0
	// Weighted sum = (1 * 0.25) + (0 * 1.0) = 0.25
	// Total weight = 0.25 + 1.0 = 1.25
	// Raw score = 0.25 / 1.25 = 0.20 (20%) -> WEAK level!
	answers := []repository.MasteryAnswer{
		{
			IsCorrect:   true,
			CompletedAt: now.Add(-42 * 24 * time.Hour), // 42 days ago
		},
		{
			IsCorrect:   false,
			CompletedAt: now, // today
		},
	}

	res := scoring.ComputeTopicMastery(answers, now, nil)

	if res.Score != 0.20 {
		t.Errorf("expected score 0.20, got %f", res.Score)
	}
	if res.Level != "WEAK" {
		t.Errorf("expected level WEAK, got %s", res.Level)
	}
	if res.Accuracy != 0.50 {
		t.Errorf("expected raw accuracy 0.50, got %f", res.Accuracy)
	}
	if !res.LowConfidence {
		t.Errorf("expected lowConfidence=true for 2 attempts (< 3)")
	}
}

func TestComputeTopicMastery_StrongMastery(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	answers := []repository.MasteryAnswer{
		{IsCorrect: true, CompletedAt: now.Add(-1 * 24 * time.Hour)},
		{IsCorrect: true, CompletedAt: now.Add(-2 * 24 * time.Hour)},
		{IsCorrect: true, CompletedAt: now.Add(-3 * 24 * time.Hour)},
		{IsCorrect: true, CompletedAt: now.Add(-4 * 24 * time.Hour)},
	}

	res := scoring.ComputeTopicMastery(answers, now, nil)

	if res.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f", res.Score)
	}
	if res.Level != "STRONG" {
		t.Errorf("expected level STRONG, got %s", res.Level)
	}
	if res.LowConfidence {
		t.Errorf("expected lowConfidence=false for 4 attempts")
	}
}

func TestComputeTopicFrequency_Empty(t *testing.T) {
	res := scoring.ComputeTopicFrequency([]int{}, 2026, nil)
	if res.Score != 0 || res.Level != "NONE" || res.QuestionCount != 0 {
		t.Errorf("unexpected empty frequency result: %+v", res)
	}
}

func TestComputeTopicFrequency_HighImportance(t *testing.T) {
	// 5 distinct recent years (2022, 2023, 2024, 2025, 2026) -> saturates frequency and coverage
	years := []int{2022, 2023, 2024, 2025, 2026}
	res := scoring.ComputeTopicFrequency(years, 2026, nil)

	if res.Score < 0.6 {
		t.Errorf("expected high score >= 0.6, got %f", res.Score)
	}
	if res.Level != "HIGH" {
		t.Errorf("expected level HIGH, got %s", res.Level)
	}
	if len(res.YearsSeen) != 5 {
		t.Errorf("expected 5 distinct years, got %d", len(res.YearsSeen))
	}
}

