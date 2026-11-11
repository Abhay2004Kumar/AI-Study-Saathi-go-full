// Package repository implements data access for Study Plans and Sessions.
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

type pgxStudyPlanRepository struct {
	pool *pgxpool.Pool
}

// NewStudyPlanRepository creates a new StudyPlanRepository.
func NewStudyPlanRepository(pool *pgxpool.Pool) StudyPlanRepository {
	return &pgxStudyPlanRepository{pool: pool}
}

// Upsert replaces any existing study plan for an exam and inserts the new schedule atomically.
func (r *pgxStudyPlanRepository) Upsert(ctx context.Context, plan *model.StudyPlan, sessions []model.StudyPlanSession) (*model.StudyPlan, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting study plan transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Delete existing plan for this exam and user (cascades to sessions)
	_, err = tx.Exec(ctx,
		`DELETE FROM "StudyPlan" WHERE "examId" = $1 AND "userId" = $2`,
		plan.ExamID, plan.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("clearing existing study plan: %w", err)
	}

	// 2. Insert new StudyPlan
	plan.ID = uuid.New().String()
	plan.GeneratedAt = time.Now().UTC()

	_, err = tx.Exec(ctx,
		`INSERT INTO "StudyPlan" (id, "examId", "userId", "startDate", "endDate", "generatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		plan.ID, plan.ExamID, plan.UserID, plan.StartDate, plan.EndDate, plan.GeneratedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting study plan: %w", err)
	}

	// 3. Insert StudyPlanSession batch using pipelined pgx.Batch (single network round-trip)
	if len(sessions) > 0 {
		now := time.Now().UTC()
		batch := &pgx.Batch{}
		insertSQL := `INSERT INTO "StudyPlanSession" (
			id, "planId", "date", "topicId", "subtopicId",
			"durationMinutes", "priorityScore", reason, completed, "completedAt", "createdAt"
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

		for i := range sessions {
			s := &sessions[i]
			s.ID = uuid.New().String()
			s.PlanID = plan.ID
			s.CreatedAt = now

			batch.Queue(insertSQL,
				s.ID, s.PlanID, s.Date, s.TopicID, s.SubtopicID,
				s.DurationMinutes, s.PriorityScore, s.Reason, s.Completed, s.CompletedAt, s.CreatedAt,
			)
		}

		br := tx.SendBatch(ctx, batch)
		for i := range sessions {
			_, err := br.Exec()
			if err != nil {
				br.Close()
				return nil, fmt.Errorf("inserting study plan session %d in batch: %w", i, err)
			}
		}
		if err := br.Close(); err != nil {
			return nil, fmt.Errorf("closing study plan session batch: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing study plan transaction: %w", err)
	}

	plan.Sessions = sessions
	return plan, nil
}

// FindByExamAndUser retrieves an active plan and all its scheduled sessions with topic names.
func (r *pgxStudyPlanRepository) FindByExamAndUser(ctx context.Context, examID, userID string) (*model.StudyPlan, error) {
	plan := &model.StudyPlan{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, "examId", "userId", "startDate", "endDate", "generatedAt"
		 FROM "StudyPlan"
		 WHERE "examId" = $1 AND "userId" = $2`,
		examID, userID,
	).Scan(
		&plan.ID, &plan.ExamID, &plan.UserID, &plan.StartDate, &plan.EndDate, &plan.GeneratedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetching study plan: %w", err)
	}

	// Fetch sessions with topic details
	rows, err := r.pool.Query(ctx,
		`SELECT sps.id, sps."planId", sps."date", sps."topicId", sps."subtopicId",
		        sps."durationMinutes", sps."priorityScore", sps.reason, sps.completed, sps."completedAt", sps."createdAt",
		        t.id, t.name
		 FROM "StudyPlanSession" sps
		 JOIN "Topic" t ON sps."topicId" = t.id
		 WHERE sps."planId" = $1
		 ORDER BY sps."date" ASC, sps."createdAt" ASC`,
		plan.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching study plan sessions: %w", err)
	}
	defer rows.Close()

	var sessions []model.StudyPlanSession
	for rows.Next() {
		var s model.StudyPlanSession
		var topicID, topicName string
		err := rows.Scan(
			&s.ID, &s.PlanID, &s.Date, &s.TopicID, &s.SubtopicID,
			&s.DurationMinutes, &s.PriorityScore, &s.Reason, &s.Completed, &s.CompletedAt, &s.CreatedAt,
			&topicID, &topicName,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning study plan session: %w", err)
		}
		s.Topic = &model.Topic{ID: topicID, Name: topicName}
		sessions = append(sessions, s)
	}

	plan.Sessions = sessions
	return plan, rows.Err()
}

// UpdateSessionCompletion marks a study session as completed or pending.
func (r *pgxStudyPlanRepository) UpdateSessionCompletion(ctx context.Context, sessionID, userID string, completed bool) (*model.StudyPlanSession, error) {
	var completedAt *time.Time
	if completed {
		now := time.Now().UTC()
		completedAt = &now
	}

	s := &model.StudyPlanSession{}
	err := r.pool.QueryRow(ctx,
		`UPDATE "StudyPlanSession" sps
		 SET completed = $1, "completedAt" = $2
		 FROM "StudyPlan" sp
		 WHERE sps.id = $3 AND sps."planId" = sp.id AND sp."userId" = $4
		 RETURNING sps.id, sps."planId", sps."date", sps."topicId", sps."subtopicId",
		           sps."durationMinutes", sps."priorityScore", sps.reason, sps.completed, sps."completedAt", sps."createdAt"`,
		completed, completedAt, sessionID, userID,
	).Scan(
		&s.ID, &s.PlanID, &s.Date, &s.TopicID, &s.SubtopicID,
		&s.DurationMinutes, &s.PriorityScore, &s.Reason, &s.Completed, &s.CompletedAt, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("updating session completion: %w", err)
	}

	return s, nil
}

