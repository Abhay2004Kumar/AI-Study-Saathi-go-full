// Package scoring implements the Smooth Weighted Round Robin (SWRR) study planning engine.
// No AI is involved: allocating study sessions is a deterministic priority scheduling problem.
package scoring

import (
	"math"
	"sort"
	"time"
)

// SWRRTopicInput represents a topic's evaluated metrics for study planning.
type SWRRTopicInput struct {
	TopicID         string
	TopicName       string
	SubtopicID      *string
	MasteryScore    float64
	MasteryLevel    string // NOT_STARTED, WEAK, DEVELOPING, STRONG
	LowConfidence   bool
	ImportanceScore *float64 // PYQ historical score
}

// StudyPlanningConfig defines weights and caps for the SWRR scheduling engine.
type StudyPlanningConfig struct {
	SessionUnitMinutes           int
	MaxTopicMinutesPerDayRatio   float64
	WeaknessWeight               float64
	ImportanceWeight             float64
	DefaultImportanceWhenUnknown float64
}

// DefaultStudyPlanningConfig holds standard production weights:
// 60% weakness, 40% PYQ importance, 30-min session slots, 65% single-topic daily cap.
var DefaultStudyPlanningConfig = StudyPlanningConfig{
	SessionUnitMinutes:           30,
	MaxTopicMinutesPerDayRatio:   0.65,
	WeaknessWeight:               0.6,
	ImportanceWeight:             0.4,
	DefaultImportanceWhenUnknown: 0.5,
}

// PlannedSession is a scheduled study block on a specific topic.
type PlannedSession struct {
	TopicID         string
	TopicName       string
	SubtopicID      *string
	DurationMinutes int
	PriorityScore   float64
	Reason          string
}

// PlannedDay holds all sessions allocated for a single calendar day.
type PlannedDay struct {
	Date     time.Time
	Sessions []PlannedSession
}

// BuildStudyPlanResult contains the complete generated schedule and any excluded topics.
type BuildStudyPlanResult struct {
	Days            []PlannedDay
	SkippedTopicIDs []string
}

type swrrCandidate struct {
	topic         SWRRTopicInput
	priorityScore float64
	currentWeight float64
}

// StartOfDay strips the time component, returning midnight in the given timezone.
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func buildReason(t SWRRTopicInput) string {
	base := "Still developing"
	switch t.MasteryLevel {
	case "NOT_STARTED":
		base = "Not started yet"
	case "WEAK":
		base = "Weak area — needs focused practice"
	}

	if t.ImportanceScore != nil && *t.ImportanceScore >= 0.6 {
		base += " · frequently tested in past papers"
	}
	return base
}

// BuildStudyPlan allocates daily study hours across topics using Smooth Weighted Round Robin.
func BuildStudyPlan(
	topics []SWRRTopicInput,
	today, examDate time.Time,
	availableHoursPerDay float64,
	cfg *StudyPlanningConfig,
) BuildStudyPlanResult {
	if cfg == nil {
		cfg = &DefaultStudyPlanningConfig
	}

	start := StartOfDay(today)
	end := StartOfDay(examDate)
	daysUntilExam := int(math.Round(end.Sub(start).Hours()/24.0)) + 1
	if daysUntilExam < 1 {
		daysUntilExam = 1
	}

	var skippedTopicIDs []string
	var scored []*swrrCandidate

	for _, t := range topics {
		// Topics confidently mastered (STRONG and not low-confidence) are excluded
		// from first-pass study plans — spaced repetition in Module 8 handles them.
		if t.MasteryLevel == "STRONG" && !t.LowConfidence {
			skippedTopicIDs = append(skippedTopicIDs, t.TopicID)
			continue
		}

		weakness := 1.0 - t.MasteryScore
		importance := cfg.DefaultImportanceWhenUnknown
		if t.ImportanceScore != nil {
			importance = *t.ImportanceScore
		}

		priorityScore := math.Round((cfg.WeaknessWeight*weakness+cfg.ImportanceWeight*importance)*100) / 100
		scored = append(scored, &swrrCandidate{
			topic:         t,
			priorityScore: priorityScore,
			currentWeight: 0,
		})
	}

	if len(scored) == 0 {
		return BuildStudyPlanResult{
			Days:            []PlannedDay{},
			SkippedTopicIDs: skippedTopicIDs,
		}
	}

	// Stable sort by priorityScore descending, then topic name
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].priorityScore != scored[j].priorityScore {
			return scored[i].priorityScore > scored[j].priorityScore
		}
		return scored[i].topic.TopicName < scored[j].topic.TopicName
	})

	var totalWeight float64
	for _, t := range scored {
		totalWeight += t.priorityScore
	}
	if totalWeight <= 0 {
		totalWeight = float64(len(scored))
	}

	dailyBudgetMinutes := int(math.Round(availableHoursPerDay * 60))
	perTopicDailyCap := int(math.Floor(float64(dailyBudgetMinutes) * cfg.MaxTopicMinutesPerDayRatio))
	if perTopicDailyCap < cfg.SessionUnitMinutes {
		perTopicDailyCap = cfg.SessionUnitMinutes
	}

	var days []PlannedDay

	for d := 0; d < daysUntilExam; d++ {
		dayDate := start.AddDate(0, 0, d)
		remaining := dailyBudgetMinutes
		usedToday := make(map[string]int, len(scored))
		var sessions []PlannedSession

		for remaining >= cfg.SessionUnitMinutes {
			// Find candidates that haven't hit today's cap
			var candidates []*swrrCandidate
			for _, t := range scored {
				if usedToday[t.topic.TopicID] < perTopicDailyCap {
					candidates = append(candidates, t)
				}
			}
			if len(candidates) == 0 {
				break
			}

			// Advance weights for eligible candidates
			for _, t := range candidates {
				increment := t.priorityScore
				if increment <= 0 {
					increment = 0.01 // non-zero increment so lower priority still rotates
				}
				t.currentWeight += increment
			}

			// Pick candidate with highest currentWeight
			var picked *swrrCandidate
			for _, t := range candidates {
				if picked == nil || t.currentWeight > picked.currentWeight {
					picked = t
				}
			}

			// Decrement chosen candidate by totalWeight (classic SWRR)
			picked.currentWeight -= totalWeight

			usedToday[picked.topic.TopicID] += cfg.SessionUnitMinutes
			remaining -= cfg.SessionUnitMinutes

			// Merge duration if already picked today, or append new session
			merged := false
			for i := range sessions {
				if sessions[i].TopicID == picked.topic.TopicID {
					sessions[i].DurationMinutes += cfg.SessionUnitMinutes
					merged = true
					break
				}
			}
			if !merged {
				sessions = append(sessions, PlannedSession{
					TopicID:         picked.topic.TopicID,
					TopicName:       picked.topic.TopicName,
					SubtopicID:      picked.topic.SubtopicID,
					DurationMinutes: cfg.SessionUnitMinutes,
					PriorityScore:   picked.priorityScore,
					Reason:          buildReason(picked.topic),
				})
			}
		}

		days = append(days, PlannedDay{
			Date:     dayDate,
			Sessions: sessions,
		})
	}

	return BuildStudyPlanResult{
		Days:            days,
		SkippedTopicIDs: skippedTopicIDs,
	}
}

