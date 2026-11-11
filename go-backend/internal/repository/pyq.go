// Package repository provides data access for Previous Year Questions (PYQs).
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxPyqRepository struct {
	pool *pgxpool.Pool
}

// NewPyqRepository creates a new PyqRepository.
func NewPyqRepository(pool *pgxpool.Pool) PyqRepository {
	return &pgxPyqRepository{pool: pool}
}

// FindByTopic retrieves all PYQ questions mapped to a topic for this student.
func (r *pgxPyqRepository) FindByTopic(ctx context.Context, topicID, userID string, excludeRejected bool) ([]PyqWithRelations, error) {
	query := `
		SELECT pq.id, pq."documentId", pq."topicId", pq."subtopicId", pq."questionText",
		       pq.options, pq."correctAnswer", pq.year, pq.confidence, pq.status,
		       d.id, d.title,
		       st.id, st.name
		FROM "PyqQuestion" pq
		JOIN "Document" d ON pq."documentId" = d.id
		LEFT JOIN "Subtopic" st ON pq."subtopicId" = st.id
		WHERE pq."topicId" = $1 AND d."userId" = $2
	`
	if excludeRejected {
		query += ` AND pq.status != 'REJECTED'`
	}
	query += ` ORDER BY pq.year DESC NULLS LAST, pq."createdAt" ASC`

	rows, err := r.pool.Query(ctx, query, topicID, userID)
	if err != nil {
		return nil, fmt.Errorf("querying topic pyq questions: %w", err)
	}
	defer rows.Close()

	var results []PyqWithRelations
	for rows.Next() {
		var item PyqWithRelations
		var subID *string
		var subName *string

		err := rows.Scan(
			&item.Question.ID,
			&item.Question.DocumentID,
			&item.Question.TopicID,
			&item.Question.SubtopicID,
			&item.Question.QuestionText,
			&item.Question.Options,
			&item.Question.CorrectAnswer,
			&item.Question.Year,
			&item.Question.Confidence,
			&item.Question.Status,
			&item.DocumentID,
			&item.DocumentTitle,
			&subID,
			&subName,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning pyq question: %w", err)
		}
		item.SubtopicID = subID
		item.SubtopicName = subName

		results = append(results, item)
	}

	return results, rows.Err()
}

// FindByExam groups all non-rejected PYQ questions under an exam by topicID.
func (r *pgxPyqRepository) FindByExam(ctx context.Context, examID string, excludeRejected bool) (map[string][]model.PyqQuestion, error) {
	query := `
		SELECT pq.id, pq."documentId", pq."topicId", pq."subtopicId", pq."questionText",
		       pq.options, pq."correctAnswer", pq.year, pq.confidence, pq.status
		FROM "PyqQuestion" pq
		JOIN "Topic" t ON pq."topicId" = t.id
		JOIN "Subject" s ON t."subjectId" = s.id
		WHERE s."examId" = $1 AND pq."topicId" IS NOT NULL
	`
	if excludeRejected {
		query += ` AND pq.status != 'REJECTED'`
	}

	rows, err := r.pool.Query(ctx, query, examID)
	if err != nil {
		return nil, fmt.Errorf("querying exam pyq questions: %w", err)
	}
	defer rows.Close()

	byTopic := make(map[string][]model.PyqQuestion)
	for rows.Next() {
		var q model.PyqQuestion
		err := rows.Scan(
			&q.ID,
			&q.DocumentID,
			&q.TopicID,
			&q.SubtopicID,
			&q.QuestionText,
			&q.Options,
			&q.CorrectAnswer,
			&q.Year,
			&q.Confidence,
			&q.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning exam pyq row: %w", err)
		}
		if q.TopicID != nil {
			byTopic[*q.TopicID] = append(byTopic[*q.TopicID], q)
		}
	}

	return byTopic, rows.Err()
}

// FindByID retrieves a single PYQ question verifying document ownership.
func (r *pgxPyqRepository) FindByID(ctx context.Context, id, userID string) (*model.PyqQuestion, error) {
	q := &model.PyqQuestion{}

	err := r.pool.QueryRow(ctx,
		`SELECT pq.id, pq."documentId", pq."topicId", pq."subtopicId", pq."questionText",
		        pq.options, pq."correctAnswer", pq.year, pq.confidence, pq.status
		 FROM "PyqQuestion" pq
		 JOIN "Document" d ON pq."documentId" = d.id
		 WHERE pq.id = $1 AND d."userId" = $2`,
		id, userID,
	).Scan(
		&q.ID,
		&q.DocumentID,
		&q.TopicID,
		&q.SubtopicID,
		&q.QuestionText,
		&q.Options,
		&q.CorrectAnswer,
		&q.Year,
		&q.Confidence,
		&q.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetching pyq question by id: %w", err)
	}
	return q, nil
}

// UpdateStatus updates the review status of a PYQ question.
func (r *pgxPyqRepository) UpdateStatus(ctx context.Context, id, status string) (*model.PyqQuestion, error) {
	q := &model.PyqQuestion{}

	err := r.pool.QueryRow(ctx,
		`UPDATE "PyqQuestion"
		 SET status = $1
		 WHERE id = $2
		 RETURNING id, "documentId", "topicId", "subtopicId", "questionText", options, "correctAnswer", year, confidence, status`,
		status, id,
	).Scan(
		&q.ID,
		&q.DocumentID,
		&q.TopicID,
		&q.SubtopicID,
		&q.QuestionText,
		&q.Options,
		&q.CorrectAnswer,
		&q.Year,
		&q.Confidence,
		&q.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("updating pyq question status: %w", err)
	}
	return q, nil
}

// FindByDocument fetches all questions extracted from a document.
func (r *pgxPyqRepository) FindByDocument(ctx context.Context, documentID string) ([]model.PyqQuestion, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, "documentId", "topicId", "subtopicId", "questionText", options, "correctAnswer", year, confidence, status
		 FROM "PyqQuestion"
		 WHERE "documentId" = $1
		 ORDER BY "createdAt" ASC`,
		documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching document pyq questions: %w", err)
	}
	defer rows.Close()

	var questions []model.PyqQuestion
	for rows.Next() {
		var q model.PyqQuestion
		err := rows.Scan(
			&q.ID,
			&q.DocumentID,
			&q.TopicID,
			&q.SubtopicID,
			&q.QuestionText,
			&q.Options,
			&q.CorrectAnswer,
			&q.Year,
			&q.Confidence,
			&q.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning pyq question: %w", err)
		}
		questions = append(questions, q)
	}
	return questions, rows.Err()
}

// ReplaceForDocument replaces all PYQ questions for a document in a transaction.
func (r *pgxPyqRepository) ReplaceForDocument(ctx context.Context, documentID string, questions []model.PyqQuestion) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning replace pyq transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM "PyqQuestion" WHERE "documentId" = $1`, documentID)
	if err != nil {
		return fmt.Errorf("clearing existing pyq questions: %w", err)
	}

	now := time.Now().UTC()
	for i := range questions {
		q := &questions[i]
		if q.ID == "" {
			q.ID = uuid.New().String()
		}
		var optsJSON []byte
		if q.Options != nil {
			optsJSON = *q.Options
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO "PyqQuestion" (id, "documentId", "topicId", "subtopicId", "questionText", "normalizedText", options, "correctAnswer", year, confidence, status, "createdAt")
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			q.ID, documentID, q.TopicID, q.SubtopicID, q.QuestionText, q.NormalizedText, optsJSON, q.CorrectAnswer, q.Year, q.Confidence, q.Status, now,
		)
		if err != nil {
			return fmt.Errorf("inserting pyq question %d: %w", i, err)
		}
	}

	return tx.Commit(ctx)
}

