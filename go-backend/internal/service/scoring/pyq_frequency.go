// Package scoring implements pure, deterministic mathematical scoring algorithms.
package scoring

import (
	"math"
	"sort"

	"github.com/aiStudySaathi/backend/internal/dto"
)

// FrequencyWeights defines the relative contribution of each signal.
type FrequencyWeights struct {
	Frequency float64
	Recency   float64
	Coverage  float64
}

// PyqFrequencyConfig holds parameters for historical importance computation.
type PyqFrequencyConfig struct {
	FrequencyCap         float64
	CoverageCapYears     float64
	RecencyHalfLifeYears float64
	Weights              FrequencyWeights
	HighThreshold        float64
	MediumThreshold      float64
}

// DefaultPyqFrequencyConfig holds standard weights: 30% volume, 40% recency, 30% coverage.
var DefaultPyqFrequencyConfig = PyqFrequencyConfig{
	FrequencyCap:         5.0,
	CoverageCapYears:     5.0,
	RecencyHalfLifeYears: 3.0,
	Weights: FrequencyWeights{
		Frequency: 0.3,
		Recency:   0.4,
		Coverage:  0.3,
	},
	HighThreshold:   0.6,
	MediumThreshold: 0.3,
}

// ComputeTopicFrequency calculates the historical importance score of a topic from previous exam years.
func ComputeTopicFrequency(years []int, currentYear int, cfg *PyqFrequencyConfig) dto.PyqFrequencyResult {
	if cfg == nil {
		cfg = &DefaultPyqFrequencyConfig
	}

	if len(years) == 0 {
		return dto.PyqFrequencyResult{
			Score:         0,
			Level:         "NONE",
			YearsSeen:     []int{},
			QuestionCount: 0,
		}
	}

	// 1. Deduplicate and sort years
	yearSet := make(map[int]bool, len(years))
	for _, y := range years {
		yearSet[y] = true
	}
	uniqueYears := make([]int, 0, len(yearSet))
	for y := range yearSet {
		uniqueYears = append(uniqueYears, y)
	}
	sort.Ints(uniqueYears)

	// 2. Frequency component = min(questionCount / frequencyCap, 1)
	frequencyComponent := math.Min(float64(len(years))/cfg.FrequencyCap, 1.0)

	// 3. Coverage component = min(uniqueYears / coverageCapYears, 1)
	var coverageComponent float64
	if len(uniqueYears) > 0 {
		coverageComponent = math.Min(float64(len(uniqueYears))/cfg.CoverageCapYears, 1.0)
	}

	// 4. Recency component = avg(0.5^(age / recencyHalfLifeYears))
	var recencyComponent float64
	if len(uniqueYears) > 0 {
		var sumRecency float64
		for _, y := range uniqueYears {
			age := float64(currentYear - y)
			if age < 0 {
				age = 0
			}
			sumRecency += math.Pow(0.5, age/cfg.RecencyHalfLifeYears)
		}
		recencyComponent = sumRecency / float64(len(uniqueYears))
	}

	// 5. Blended weighted score
	rawScore := cfg.Weights.Frequency*frequencyComponent +
		cfg.Weights.Recency*recencyComponent +
		cfg.Weights.Coverage*coverageComponent
	score := math.Round(rawScore*100) / 100

	// 6. Level classification
	var level string
	if score >= cfg.HighThreshold {
		level = "HIGH"
	} else if score >= cfg.MediumThreshold {
		level = "MEDIUM"
	} else {
		level = "LOW"
	}

	return dto.PyqFrequencyResult{
		Score:         score,
		Level:         level,
		YearsSeen:     uniqueYears,
		QuestionCount: len(years),
	}
}

