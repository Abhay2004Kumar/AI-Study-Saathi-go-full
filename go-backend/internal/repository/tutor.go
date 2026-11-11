// Package repository implements data access for Tutoring sessions.
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

type pgxTutorSessionRepository struct {
	pool *pgxpool.Pool
}

// NewTutorSessionRepository creates a new TutorSessionRepository.
func NewTutorSessionRepository(pool *pgxpool.Pool) TutorSessionRepository {
	return &pgxTutorSessionRepository{pool: pool}
}

// Create stores a new tutoring session.
func (r *pgxTutorSessionRepository) Create(ctx context.Context, session *model.TutorSession) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	session.CreatedAt = now
	session.UpdatedAt = now

	_, err := r.pool.Exec(ctx,
		`INSERT INTO "TutorSession" (
			id, "userId", subject, topic, status,
			"conversationHistory", "currentConcept", "weakAreas", "questionCount",
			"createdAt", "updatedAt"
		 ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		session.ID, session.UserID, session.Subject, session.Topic, session.Status,
		session.ConversationHistory, session.CurrentConcept, session.WeakAreas, session.QuestionCount,
		session.CreatedAt, session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting tutor session: %w", err)
	}

	return nil
}

// FindByID retrieves a tutor session by ID verifying ownership.
func (r *pgxTutorSessionRepository) FindByID(ctx context.Context, id, userID string) (*model.TutorSession, error) {
	s := &model.TutorSession{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, "userId", subject, topic, status,
		        "conversationHistory", "currentConcept", "weakAreas", "questionCount",
		        "createdAt", "updatedAt"
		 FROM "TutorSession"
		 WHERE id = $1 AND "userId" = $2`,
		id, userID,
	).Scan(
		&s.ID, &s.UserID, &s.Subject, &s.Topic, &s.Status,
		&s.ConversationHistory, &s.CurrentConcept, &s.WeakAreas, &s.QuestionCount,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetching tutor session: %w", err)
	}

	return s, nil
}

// Update modifies dynamic tutor session fields (status, currentConcept, conversationHistory, weakAreas, questionCount).
func (r *pgxTutorSessionRepository) Update(ctx context.Context, id string, updates map[string]interface{}) (*model.TutorSession, error) {
	status := updates["status"].(string)
	currentConcept, _ := updates["currentConcept"].(*string)
	conversationHistory := updates["conversationHistory"].([]byte)
	weakAreas := updates["weakAreas"].([]byte)
	questionCount := updates["questionCount"].(int)
	now := time.Now().UTC()

	s := &model.TutorSession{}
	err := r.pool.QueryRow(ctx,
		`UPDATE "TutorSession"
		 SET status = $1, "currentConcept" = $2, "conversationHistory" = $3, "weakAreas" = $4, "questionCount" = $5, "updatedAt" = $6
		 WHERE id = $7
		 RETURNING id, "userId", subject, topic, status,
		           "conversationHistory", "currentConcept", "weakAreas", "questionCount",
		           "createdAt", "updatedAt"`,
		status, currentConcept, conversationHistory, weakAreas, questionCount, now, id,
	).Scan(
		&s.ID, &s.UserID, &s.Subject, &s.Topic, &s.Status,
		&s.ConversationHistory, &s.CurrentConcept, &s.WeakAreas, &s.QuestionCount,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("updating tutor session: %w", err)
	}

	return s, nil
}

