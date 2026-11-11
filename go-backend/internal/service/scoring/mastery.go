// Package scoring implements pure, deterministic mathematical scoring algorithms.
// There is intentionally NO AI/LLM here: mastery accuracy and historical PYQ importance
// are strict mathematical facts derived from database events.
package scoring

import (
	"math"
	"time"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/repository"
)

// MasteryConfig holds tuneable parameters for the recency-weighted accuracy algorithm.
type MasteryConfig struct {
	RecencyHalfLifeDays      float64
	MinAttemptsForConfidence int
	WeakThreshold            float64
	StrongThreshold          float64
}

// DefaultMasteryConfig represents standard half-life decay (21 days) and thresholds.
var DefaultMasteryConfig = MasteryConfig{
	RecencyHalfLifeDays:      21.0,
	MinAttemptsForConfidence: 3,
	WeakThreshold:            0.5,
	StrongThreshold:          0.8,
}

// ComputeTopicMastery evaluates a student's mastery across QuizAttemptAnswer rows.
// Recent attempts contribute exponentially more than older attempts using half-life decay.
func ComputeTopicMastery(answers []repository.MasteryAnswer, now time.Time, cfg *MasteryConfig) dto.MasteryResult {
	if cfg == nil {
		cfg = &DefaultMasteryConfig
	}

	if len(answers) == 0 {
		return dto.MasteryResult{
			Score:         0,
			Level:         "NOT_STARTED",
			Accuracy:      0,
			AttemptCount:  0,
			LowConfidence: true,
		}
	}

	attemptCount := len(answers)
	correctCount := 0
	for _, a := range answers {
		if a.IsCorrect {
			correctCount++
		}
	}

	accuracy := math.Round((float64(correctCount)/float64(attemptCount))*100) / 100

	var weightedSum float64
	var weightTotal float64

	for _, a := range answers {
		ageDuration := now.Sub(a.CompletedAt)
		ageDays := ageDuration.Hours() / 24.0
		if ageDays < 0 {
			ageDays = 0
		}
		weight := math.Pow(0.5, ageDays/cfg.RecencyHalfLifeDays)

		if a.IsCorrect {
			weightedSum += weight
		}
		weightTotal += weight
	}

	var rawScore float64
	if weightTotal > 0 {
		rawScore = weightedSum / weightTotal
	}
	score := math.Round(rawScore*100) / 100

	var level string
	if score >= cfg.StrongThreshold {
		level = "STRONG"
	} else if score >= cfg.WeakThreshold {
		level = "DEVELOPING"
	} else {
		level = "WEAK"
	}

	return dto.MasteryResult{
		Score:         score,
		Level:         level,
		Accuracy:      accuracy,
		AttemptCount:  attemptCount,
		LowConfidence: attemptCount < cfg.MinAttemptsForConfidence,
	}
}

