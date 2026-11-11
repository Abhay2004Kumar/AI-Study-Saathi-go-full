// Package repository provides database queries for Mastery and PYQ data.
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxMasteryRepository struct {
	pool *pgxpool.Pool
}

// NewMasteryRepository creates a new MasteryRepository.
func NewMasteryRepository(pool *pgxpool.Pool) MasteryRepository {
	return &pgxMasteryRepository{pool: pool}
}

// FindAnswersByTopic retrieves all student answers for a specific topic where the quiz attempt was completed.
func (r *pgxMasteryRepository) FindAnswersByTopic(ctx context.Context, userID, topicID string) ([]MasteryAnswer, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT qaa."isCorrect", qa."completedAt"
		 FROM "QuizAttemptAnswer" qaa
		 JOIN "QuizAttempt" qa ON qaa."attemptId" = qa.id
		 JOIN "QuizQuestion" qq ON qaa."questionId" = qq.id
		 JOIN "Quiz" q ON qq."quizId" = q.id
		 WHERE q."topicId" = $1 AND qa."userId" = $2 AND qa."completedAt" IS NOT NULL
		 ORDER BY qa."completedAt" ASC`,
		topicID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying topic mastery answers: %w", err)
	}
	defer rows.Close()

	var answers []MasteryAnswer
	for rows.Next() {
		var a MasteryAnswer
		if err := rows.Scan(&a.IsCorrect, &a.CompletedAt); err != nil {
			return nil, fmt.Errorf("scanning mastery answer: %w", err)
		}
		a.TopicID = topicID
		answers = append(answers, a)
	}

	return answers, rows.Err()
}

// FindAnswersByExam retrieves answers across ALL topics under an exam for the student in one query.
func (r *pgxMasteryRepository) FindAnswersByExam(ctx context.Context, userID, examID string) (map[string][]MasteryAnswer, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT qaa."isCorrect", qa."completedAt", q."topicId"
		 FROM "QuizAttemptAnswer" qaa
		 JOIN "QuizAttempt" qa ON qaa."attemptId" = qa.id
		 JOIN "QuizQuestion" qq ON qaa."questionId" = qq.id
		 JOIN "Quiz" q ON qq."quizId" = q.id
		 JOIN "Topic" t ON q."topicId" = t.id
		 JOIN "Subject" s ON t."subjectId" = s.id
		 WHERE s."examId" = $1 AND qa."userId" = $2 AND qa."completedAt" IS NOT NULL
		 ORDER BY qa."completedAt" ASC`,
		examID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying exam mastery answers: %w", err)
	}
	defer rows.Close()

	byTopic := make(map[string][]MasteryAnswer)
	for rows.Next() {
		var a MasteryAnswer
		var topicID string
		if err := rows.Scan(&a.IsCorrect, &a.CompletedAt, &topicID); err != nil {
			return nil, fmt.Errorf("scanning exam mastery answer: %w", err)
		}
		a.TopicID = topicID
		byTopic[topicID] = append(byTopic[topicID], a)
	}

	return byTopic, rows.Err()
}

