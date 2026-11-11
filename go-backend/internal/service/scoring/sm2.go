// Package scoring implements the SuperMemo SM-2 spaced repetition algorithm.
// No AI is involved: review scheduling is deterministic mathematical spacing.
package scoring

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

// SM-2 Constants
const (
	MinEaseFactor     = 1.3
	DefaultEaseFactor = 2.5
)

// GradeQuality maps 4 Anki-style review buttons to SM-2 quality grades (0-5 scale).
var GradeQuality = map[string]int{
	"AGAIN": 0,
	"HARD":  3,
	"GOOD":  4,
	"EASY":  5,
}

// SM2State represents the current repetition state of a flashcard.
type SM2State struct {
	EaseFactor   float64
	IntervalDays int
	Repetitions  int
}

// SM2Result is the updated repetition state after applying a review grade.
type SM2Result struct {
	EaseFactor     float64
	IntervalDays   int
	Repetitions    int
	DueAt          time.Time
	LastReviewedAt time.Time
}

// ApplyReview calculates the next due date and interval using the SM-2 algorithm.
func ApplyReview(card SM2State, grade string, now time.Time) (*SM2Result, error) {
	quality, ok := GradeQuality[grade]
	if !ok {
		return nil, fmt.Errorf("unknown grade: %s", grade)
	}

	currentEF := card.EaseFactor
	if currentEF <= 0 {
		currentEF = DefaultEaseFactor
	}

	// Calculate new ease factor:
	// EF' = EF + (0.1 - (5 - q) * (0.08 + (5 - q) * 0.02))
	qDiff := float64(5 - quality)
	newEF := currentEF + (0.1 - qDiff*(0.08+qDiff*0.02))
	if newEF < MinEaseFactor {
		newEF = MinEaseFactor
	}
	easeFactor := math.Round(newEF*100) / 100

	var repetitions int
	var intervalDays int

	if quality < 3 {
		// Forgotten (AGAIN) — reset repetitions to 0 and interval to 1 day.
		// The ease factor penalty is still retained so repeatedly forgotten cards
		// progress more slowly upon recovery.
		repetitions = 0
		intervalDays = 1
	} else {
		// Successful recall (HARD, GOOD, EASY)
		repetitions = card.Repetitions + 1
		if repetitions == 1 {
			intervalDays = 1
		} else if repetitions == 2 {
			intervalDays = 6
		} else {
			baseInterval := card.IntervalDays
			if baseInterval <= 0 {
				baseInterval = 6
			}
			intervalDays = int(math.Round(float64(baseInterval) * easeFactor))
		}
	}

	dueAt := now.Add(time.Duration(intervalDays) * 24 * time.Hour)

	return &SM2Result{
		EaseFactor:     easeFactor,
		IntervalDays:   intervalDays,
		Repetitions:    repetitions,
		DueAt:          dueAt,
		LastReviewedAt: now,
	}, nil
}

var multiSpaceRegex = regexp.MustCompile(`\s+`)

// NormalizeFrontText standardizes card front text for deduplication:
// trims whitespace, lowercases, and collapses multiple spaces into one.
func NormalizeFrontText(text string) string {
	trimmed := strings.TrimSpace(text)
	lowered := strings.ToLower(trimmed)
	return multiSpaceRegex.ReplaceAllString(lowered, " ")
}

// RawFlashcard holds the prompt output from the LLM before persistence.
type RawFlashcard struct {
	Front string `json:"front"`
	Back  string `json:"back"`
	Hint  string `json:"hint"`
}

// DedupeFlashcards drops any card whose normalized front matches an existing card
// in the database or an earlier card in the same generation batch.
func DedupeFlashcards(cards []RawFlashcard, existingSet map[string]bool) []RawFlashcard {
	seen := make(map[string]bool, len(existingSet)+len(cards))
	for k, v := range existingSet {
		seen[k] = v
	}

	var result []RawFlashcard
	for _, c := range cards {
		if strings.TrimSpace(c.Front) == "" {
			continue
		}
		norm := NormalizeFrontText(c.Front)
		if seen[norm] {
			continue
		}
		seen[norm] = true
		result = append(result, c)
	}

	return result
}

