// Package repository implements data access for AI Assistant and Tutoring sessions.
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

type pgxAssistantSessionRepository struct {
	pool *pgxpool.Pool
}

// NewAssistantSessionRepository creates a new AssistantSessionRepository.
func NewAssistantSessionRepository(pool *pgxpool.Pool) AssistantSessionRepository {
	return &pgxAssistantSessionRepository{pool: pool}
}

// FindByExamAndUser retrieves an existing assistant session for a specific exam and user.
func (r *pgxAssistantSessionRepository) FindByExamAndUser(ctx context.Context, examID, userID string) (*model.AssistantSession, error) {
	s := &model.AssistantSession{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, "userId", "examId", messages, "createdAt", "updatedAt"
		 FROM "AssistantSession"
		 WHERE "examId" = $1 AND "userId" = $2`,
		examID, userID,
	).Scan(
		&s.ID, &s.UserID, &s.ExamID, &s.Messages, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetching assistant session: %w", err)
	}
	return s, nil
}

// Upsert creates or appends conversation messages to an exam's assistant session.
func (r *pgxAssistantSessionRepository) Upsert(ctx context.Context, examID, userID string, messages []byte) (*model.AssistantSession, error) {
	s := &model.AssistantSession{}
	now := time.Now().UTC()
	newID := uuid.New().String()

	err := r.pool.QueryRow(ctx,
		`INSERT INTO "AssistantSession" (id, "userId", "examId", messages, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT ("examId", "userId")
		 DO UPDATE SET messages = EXCLUDED.messages, "updatedAt" = EXCLUDED."updatedAt"
		 RETURNING id, "userId", "examId", messages, "createdAt", "updatedAt"`,
		newID, userID, examID, messages, now, now,
	).Scan(
		&s.ID, &s.UserID, &s.ExamID, &s.Messages, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upserting assistant session: %w", err)
	}

	return s, nil
}

