package service_test

import (
	"testing"
	"time"

	"github.com/aiStudySaathi/backend/internal/service/scoring"
)

func TestBuildStudyPlan_MasteredTopicsExcluded(t *testing.T) {
	today := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	examDate := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC) // 3 days

	impHigh := 0.8
	topics := []scoring.SWRRTopicInput{
		{
			TopicID:         "top-mastered",
			TopicName:       "Basic Logic Gates",
			MasteryScore:    0.95,
			MasteryLevel:    "STRONG",
			LowConfidence:   false, // Excluded!
			ImportanceScore: &impHigh,
		},
		{
			TopicID:         "top-weak",
			TopicName:       "Pipelining Hazards",
			MasteryScore:    0.20,
			MasteryLevel:    "WEAK",
			LowConfidence:   false,
			ImportanceScore: &impHigh,
		},
	}

	result := scoring.BuildStudyPlan(topics, today, examDate, 2.0, nil)

	if len(result.SkippedTopicIDs) != 1 || result.SkippedTopicIDs[0] != "top-mastered" {
		t.Errorf("expected top-mastered in SkippedTopicIDs, got %v", result.SkippedTopicIDs)
	}

	if len(result.Days) != 3 {
		t.Fatalf("expected 3 days, got %d", len(result.Days))
	}

	// Verify only top-weak is scheduled
	for _, day := range result.Days {
		for _, sess := range day.Sessions {
			if sess.TopicID != "top-weak" {
				t.Errorf("expected only top-weak scheduled, got %s", sess.TopicID)
			}
		}
	}
}

func TestBuildStudyPlan_DailyCapEnforcement(t *testing.T) {
	today := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	examDate := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC) // 1 day

	imp := 0.7
	// 2 hours = 120 minutes daily budget.
	// 65% cap = floor(120 * 0.65) = 78 minutes -> max 60 minutes (two 30-min units).
	topics := []scoring.SWRRTopicInput{
		{
			TopicID:         "top-1",
			TopicName:       "Virtual Memory",
			MasteryScore:    0.10,
			MasteryLevel:    "WEAK",
			ImportanceScore: &imp,
		},
		{
			TopicID:         "top-2",
			TopicName:       "CPU Scheduling",
			MasteryScore:    0.10,
			MasteryLevel:    "WEAK",
			ImportanceScore: &imp,
		},
	}

	result := scoring.BuildStudyPlan(topics, today, examDate, 2.0, nil)

	if len(result.Days) != 1 {
		t.Fatalf("expected 1 day plan")
	}

	sessions := result.Days[0].Sessions
	for _, s := range sessions {
		if s.DurationMinutes > 78 {
			t.Errorf("session duration %d exceeded 65%% cap (78 mins)", s.DurationMinutes)
		}
	}
}

func TestBuildStudyPlan_ReasonStringConstruction(t *testing.T) {
	today := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	examDate := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)

	highImp := 0.85
	lowImp := 0.20
	topics := []scoring.SWRRTopicInput{
		{
			TopicID:         "top-weak-high",
			TopicName:       "Concurrency",
			MasteryLevel:    "WEAK",
			ImportanceScore: &highImp,
		},
		{
			TopicID:         "top-dev-low",
			TopicName:       "File Systems",
			MasteryLevel:    "DEVELOPING",
			ImportanceScore: &lowImp,
		},
	}

	result := scoring.BuildStudyPlan(topics, today, examDate, 2.0, nil)
	sessions := result.Days[0].Sessions

	foundWeakHigh := false
	foundDevLow := false
	for _, s := range sessions {
		if s.TopicID == "top-weak-high" {
			foundWeakHigh = true
			expected := "Weak area — needs focused practice · frequently tested in past papers"
			if s.Reason != expected {
				t.Errorf("expected reason '%s', got '%s'", expected, s.Reason)
			}
		}
		if s.TopicID == "top-dev-low" {
			foundDevLow = true
			expected := "Still developing"
			if s.Reason != expected {
				t.Errorf("expected reason '%s', got '%s'", expected, s.Reason)
			}
		}
	}

	if !foundWeakHigh || !foundDevLow {
		t.Errorf("missing scheduled sessions for topics")
	}
}

