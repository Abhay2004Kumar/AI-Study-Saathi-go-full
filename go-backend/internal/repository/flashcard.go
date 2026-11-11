// Package repository implements data access for Flashcards and SM-2 state.
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

type pgxFlashcardRepository struct {
	pool *pgxpool.Pool
}

// NewFlashcardRepository creates a new FlashcardRepository.
func NewFlashcardRepository(pool *pgxpool.Pool) FlashcardRepository {
	return &pgxFlashcardRepository{pool: pool}
}

// CreateMany inserts a batch of flashcards into the database.
func (r *pgxFlashcardRepository) CreateMany(ctx context.Context, cards []model.Flashcard) error {
	if len(cards) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("starting flashcard batch transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	for i := range cards {
		c := &cards[i]
		if c.ID == "" {
			c.ID = uuid.New().String()
		}
		if c.EaseFactor <= 0 {
			c.EaseFactor = 2.5
		}
		if c.DueAt.IsZero() {
			c.DueAt = now
		}
		c.CreatedAt = now
		c.UpdatedAt = now

		_, err = tx.Exec(ctx,
			`INSERT INTO "Flashcard" (
				id, "userId", "subjectId", "topicId", "subtopicId",
				front, back, hint, "normalizedFront", "sourceDocumentIds",
				"easeFactor", "intervalDays", repetitions, "dueAt", "lastReviewedAt",
				"createdAt", "updatedAt"
			 ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
			c.ID, c.UserID, c.SubjectID, c.TopicID, c.SubtopicID,
			c.Front, c.Back, c.Hint, c.NormalizedFront, c.SourceDocumentIDs,
			c.EaseFactor, c.IntervalDays, c.Repetitions, c.DueAt, c.LastReviewedAt,
			c.CreatedAt, c.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("inserting flashcard %d: %w", i, err)
		}
	}

	return tx.Commit(ctx)
}

// FindByTopic retrieves all cards in a topic's deck, ordered by creation time.
func (r *pgxFlashcardRepository) FindByTopic(ctx context.Context, topicID, userID string) ([]model.Flashcard, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, "userId", "subjectId", "topicId", "subtopicId",
		        front, back, hint, "normalizedFront", "sourceDocumentIds",
		        "easeFactor", "intervalDays", repetitions, "dueAt", "lastReviewedAt",
		        "createdAt", "updatedAt"
		 FROM "Flashcard"
		 WHERE "topicId" = $1 AND "userId" = $2
		 ORDER BY "createdAt" ASC`,
		topicID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying topic flashcards: %w", err)
	}
	defer rows.Close()

	var cards []model.Flashcard
	for rows.Next() {
		var c model.Flashcard
		err := rows.Scan(
			&c.ID, &c.UserID, &c.SubjectID, &c.TopicID, &c.SubtopicID,
			&c.Front, &c.Back, &c.Hint, &c.NormalizedFront, &c.SourceDocumentIDs,
			&c.EaseFactor, &c.IntervalDays, &c.Repetitions, &c.DueAt, &c.LastReviewedAt,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning flashcard row: %w", err)
		}
		cards = append(cards, c)
	}

	return cards, rows.Err()
}

// FindDue returns cards across the entire exam whose due date has arrived (dueAt <= now).
func (r *pgxFlashcardRepository) FindDue(ctx context.Context, userID, examID string, limit int) ([]model.Flashcard, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	now := time.Now().UTC()
	rows, err := r.pool.Query(ctx,
		`SELECT f.id, f."userId", f."subjectId", f."topicId", f."subtopicId",
		        f.front, f.back, f.hint, f."normalizedFront", f."sourceDocumentIds",
		        f."easeFactor", f."intervalDays", f.repetitions, f."dueAt", f."lastReviewedAt",
		        f."createdAt", f."updatedAt"
		 FROM "Flashcard" f
		 JOIN "Topic" t ON f."topicId" = t.id
		 JOIN "Subject" s ON t."subjectId" = s.id
		 WHERE f."userId" = $1 AND s."examId" = $2 AND f."dueAt" <= $3
		 ORDER BY f."dueAt" ASC
		 LIMIT $4`,
		userID, examID, now, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("querying due flashcards: %w", err)
	}
	defer rows.Close()

	var cards []model.Flashcard
	for rows.Next() {
		var c model.Flashcard
		err := rows.Scan(
			&c.ID, &c.UserID, &c.SubjectID, &c.TopicID, &c.SubtopicID,
			&c.Front, &c.Back, &c.Hint, &c.NormalizedFront, &c.SourceDocumentIDs,
			&c.EaseFactor, &c.IntervalDays, &c.Repetitions, &c.DueAt, &c.LastReviewedAt,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning due flashcard row: %w", err)
		}
		cards = append(cards, c)
	}

	return cards, rows.Err()
}

// FindByID retrieves a single card verifying student ownership.
func (r *pgxFlashcardRepository) FindByID(ctx context.Context, id, userID string) (*model.Flashcard, error) {
	c := &model.Flashcard{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, "userId", "subjectId", "topicId", "subtopicId",
		        front, back, hint, "normalizedFront", "sourceDocumentIds",
		        "easeFactor", "intervalDays", repetitions, "dueAt", "lastReviewedAt",
		        "createdAt", "updatedAt"
		 FROM "Flashcard"
		 WHERE id = $1 AND "userId" = $2`,
		id, userID,
	).Scan(
		&c.ID, &c.UserID, &c.SubjectID, &c.TopicID, &c.SubtopicID,
		&c.Front, &c.Back, &c.Hint, &c.NormalizedFront, &c.SourceDocumentIDs,
		&c.EaseFactor, &c.IntervalDays, &c.Repetitions, &c.DueAt, &c.LastReviewedAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetching flashcard by id: %w", err)
	}
	return c, nil
}

// UpdateReview persists updated SM-2 interval and due date.
func (r *pgxFlashcardRepository) UpdateReview(ctx context.Context, id string, easeFactor float64, intervalDays, repetitions int, dueAt, lastReviewedAt time.Time) (*model.Flashcard, error) {
	c := &model.Flashcard{}
	now := time.Now().UTC()

	err := r.pool.QueryRow(ctx,
		`UPDATE "Flashcard"
		 SET "easeFactor" = $1, "intervalDays" = $2, repetitions = $3, "dueAt" = $4, "lastReviewedAt" = $5, "updatedAt" = $6
		 WHERE id = $7
		 RETURNING id, "userId", "subjectId", "topicId", "subtopicId",
		           front, back, hint, "normalizedFront", "sourceDocumentIds",
		           "easeFactor", "intervalDays", repetitions, "dueAt", "lastReviewedAt",
		           "createdAt", "updatedAt"`,
		easeFactor, intervalDays, repetitions, dueAt, lastReviewedAt, now, id,
	).Scan(
		&c.ID, &c.UserID, &c.SubjectID, &c.TopicID, &c.SubtopicID,
		&c.Front, &c.Back, &c.Hint, &c.NormalizedFront, &c.SourceDocumentIDs,
		&c.EaseFactor, &c.IntervalDays, &c.Repetitions, &c.DueAt, &c.LastReviewedAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("updating flashcard review: %w", err)
	}

	return c, nil
}

// FindNormalizedFrontsByTopic returns all existing normalized front texts for deduplication.
func (r *pgxFlashcardRepository) FindNormalizedFrontsByTopic(ctx context.Context, topicID, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT "normalizedFront"
		 FROM "Flashcard"
		 WHERE "topicId" = $1 AND "userId" = $2`,
		topicID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying normalized fronts: %w", err)
	}
	defer rows.Close()

	var fronts []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, fmt.Errorf("scanning normalized front: %w", err)
		}
		fronts = append(fronts, f)
	}

	return fronts, rows.Err()
}

