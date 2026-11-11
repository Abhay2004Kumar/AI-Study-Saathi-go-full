package service_test

import (
	"testing"
	"time"

	"github.com/aiStudySaathi/backend/internal/service/scoring"
)

func TestSM2_FirstReview(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	initial := scoring.SM2State{
		EaseFactor:   scoring.DefaultEaseFactor,
		IntervalDays: 0,
		Repetitions:  0,
	}

	// 1st Review: GOOD
	res, err := scoring.ApplyReview(initial, "GOOD", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Repetitions != 1 {
		t.Errorf("expected repetitions=1, got %d", res.Repetitions)
	}
	if res.IntervalDays != 1 {
		t.Errorf("expected interval=1 for first review, got %d", res.IntervalDays)
	}
	expectedDue := now.Add(24 * time.Hour)
	if !res.DueAt.Equal(expectedDue) {
		t.Errorf("expected dueAt=%v, got %v", expectedDue, res.DueAt)
	}
}

func TestSM2_SecondAndThirdReview(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

	// 2nd Review: GOOD -> should give interval of 6 days
	state2 := scoring.SM2State{
		EaseFactor:   2.5,
		IntervalDays: 1,
		Repetitions:  1,
	}
	res2, err := scoring.ApplyReview(state2, "GOOD", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.Repetitions != 2 || res2.IntervalDays != 6 {
		t.Errorf("expected rep=2, interval=6; got rep=%d, interval=%d", res2.Repetitions, res2.IntervalDays)
	}

	// 3rd Review: GOOD -> interval = round(6 * 2.5) = 15 days
	state3 := scoring.SM2State{
		EaseFactor:   res2.EaseFactor,
		IntervalDays: res2.IntervalDays,
		Repetitions:  res2.Repetitions,
	}
	res3, err := scoring.ApplyReview(state3, "GOOD", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res3.Repetitions != 3 || res3.IntervalDays != 15 {
		t.Errorf("expected rep=3, interval=15; got rep=%d, interval=%d", res3.Repetitions, res3.IntervalDays)
	}
}

func TestSM2_AgainResetsSchedule(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

	// Card was at 4 repetitions and 35 days interval
	state := scoring.SM2State{
		EaseFactor:   2.5,
		IntervalDays: 35,
		Repetitions:  4,
	}

	// Student clicks AGAIN (forgotten)
	res, err := scoring.ApplyReview(state, "AGAIN", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Repetitions != 0 {
		t.Errorf("expected repetitions reset to 0, got %d", res.Repetitions)
	}
	if res.IntervalDays != 1 {
		t.Errorf("expected interval reset to 1 day, got %d", res.IntervalDays)
	}
	// Ease factor should drop by 0.8: 2.5 + (0.1 - 5 * (0.08 + 5 * 0.02)) = 2.5 - 0.8 = 1.70
	if res.EaseFactor != 1.70 {
		t.Errorf("expected ease factor penalty to 1.70, got %f", res.EaseFactor)
	}
}

func TestSM2_MinEaseFactorFloor(t *testing.T) {
	now := time.Now().UTC()
	state := scoring.SM2State{
		EaseFactor:   1.4,
		IntervalDays: 1,
		Repetitions:  1,
	}

	// AGAIN should reduce EF by 0.8, but floor is 1.3
	res, _ := scoring.ApplyReview(state, "AGAIN", now)
	if res.EaseFactor != 1.3 {
		t.Errorf("expected ease factor capped at minimum 1.3, got %f", res.EaseFactor)
	}
}

func TestNormalizeAndDedupeFlashcards(t *testing.T) {
	existingSet := map[string]bool{
		"what is a process?": true,
	}

	candidates := []scoring.RawFlashcard{
		{Front: "What is a process?", Back: "An executing program."},                          // duplicate of existing
		{Front: "  WHAT  IS  A   THREAD?  ", Back: "A lightweight unit of CPU utilization."},   // should normalize
		{Front: "What is a thread?", Back: "Duplicate in same batch."},                          // duplicate within batch
		{Front: "What is paging?", Back: "Memory management scheme."},                           // unique
	}

	deduped := scoring.DedupeFlashcards(candidates, existingSet)

	if len(deduped) != 2 {
		t.Fatalf("expected exactly 2 deduped cards, got %d", len(deduped))
	}
	if scoring.NormalizeFrontText(deduped[0].Front) != "what is a thread?" {
		t.Errorf("unexpected card 0: %s", deduped[0].Front)
	}
	if scoring.NormalizeFrontText(deduped[1].Front) != "what is paging?" {
		t.Errorf("unexpected card 1: %s", deduped[1].Front)
	}
}

