// Package repository implements data access for ResourceTopicMapping.
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

type pgxTopicMappingRepository struct {
	pool *pgxpool.Pool
}

// NewTopicMappingRepository creates a new TopicMappingRepository.
func NewTopicMappingRepository(pool *pgxpool.Pool) TopicMappingRepository {
	return &pgxTopicMappingRepository{pool: pool}
}

// ReplaceForDocument replaces all mappings for a document in a batch transaction.
func (r *pgxTopicMappingRepository) ReplaceForDocument(ctx context.Context, documentID string, mappings []model.ResourceTopicMapping) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning replace topic mapping tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM "ResourceTopicMapping" WHERE "documentId" = $1`, documentID)
	if err != nil {
		return fmt.Errorf("clearing existing topic mappings: %w", err)
	}

	if len(mappings) > 0 {
		now := time.Now().UTC()
		batch := &pgx.Batch{}
		for _, m := range mappings {
			id := m.ID
			if id == "" {
				id = uuid.New().String()
			}
			batch.Queue(
				`INSERT INTO "ResourceTopicMapping" (
					id, "documentId", "topicId", "subtopicId", "startPage", "endPage", 
					confidence, status, "createdAt", "updatedAt"
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
				id, documentID, m.TopicID, m.SubtopicID, m.StartPage, m.EndPage,
				m.Confidence, m.Status, now, now,
			)
		}

		br := tx.SendBatch(ctx, batch)
		for range mappings {
			if _, err := br.Exec(); err != nil {
				br.Close()
				return fmt.Errorf("inserting topic mapping: %w", err)
			}
		}
		if err := br.Close(); err != nil {
			return fmt.Errorf("closing topic mapping batch: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// FindByDocument returns all mappings for a document, enriched with topic & subtopic names.
func (r *pgxTopicMappingRepository) FindByDocument(ctx context.Context, documentID string) ([]TopicResourceItem, error) {
	query := `
		SELECT 
			m.id, m."documentId", m."topicId", m."subtopicId", m."startPage", m."endPage", 
			m.confidence, m.status, m."createdAt",
			d.id, d.title, d.category, d."fileName", d."processingStatus",
			st.name,
			t.name
		FROM "ResourceTopicMapping" m
		JOIN "Document" d ON m."documentId" = d.id
		JOIN "Topic" t ON m."topicId" = t.id
		LEFT JOIN "Subtopic" st ON m."subtopicId" = st.id
		WHERE m."documentId" = $1
		ORDER BY m."startPage" ASC NULLS LAST, m."createdAt" ASC
	`

	rows, err := r.pool.Query(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("querying mappings for document: %w", err)
	}
	defer rows.Close()

	var items []TopicResourceItem
	for rows.Next() {
		var item TopicResourceItem
		err := rows.Scan(
			&item.ID,
			&item.DocumentID,
			&item.TopicID,
			&item.SubtopicID,
			&item.StartPage,
			&item.EndPage,
			&item.Confidence,
			&item.Status,
			&item.CreatedAt,
			&item.DocID,
			&item.DocTitle,
			&item.DocCategory,
			&item.DocFileName,
			&item.DocProcessingStatus,
			&item.SubtopicName,
			&item.TopicName,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning topic mapping row: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// FindByTopicWithRelations returns all mappings for a topic, filtered by the student's owned documents.
func (r *pgxTopicMappingRepository) FindByTopicWithRelations(ctx context.Context, topicID, userID string) ([]TopicResourceItem, error) {
	query := `
		SELECT 
			m.id, m."documentId", m."topicId", m."subtopicId", m."startPage", m."endPage", 
			m.confidence, m.status, m."createdAt",
			d.id, d.title, d.category, d."fileName", d."processingStatus",
			st.name,
			t.name
		FROM "ResourceTopicMapping" m
		JOIN "Document" d ON m."documentId" = d.id
		JOIN "Topic" t ON m."topicId" = t.id
		LEFT JOIN "Subtopic" st ON m."subtopicId" = st.id
		WHERE m."topicId" = $1 AND d."userId" = $2
		ORDER BY m."startPage" ASC NULLS LAST, m."createdAt" ASC
	`

	rows, err := r.pool.Query(ctx, query, topicID, userID)
	if err != nil {
		return nil, fmt.Errorf("querying topic mappings for topic: %w", err)
	}
	defer rows.Close()

	var items []TopicResourceItem
	for rows.Next() {
		var item TopicResourceItem
		err := rows.Scan(
			&item.ID,
			&item.DocumentID,
			&item.TopicID,
			&item.SubtopicID,
			&item.StartPage,
			&item.EndPage,
			&item.Confidence,
			&item.Status,
			&item.CreatedAt,
			&item.DocID,
			&item.DocTitle,
			&item.DocCategory,
			&item.DocFileName,
			&item.DocProcessingStatus,
			&item.SubtopicName,
			&item.TopicName,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning topic resource row: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// FindDocumentIDsByTopic finds document IDs associated with a topic.
func (r *pgxTopicMappingRepository) FindDocumentIDsByTopic(ctx context.Context, topicID string, excludeRejected bool) ([]string, error) {
	query := `SELECT DISTINCT "documentId" FROM "ResourceTopicMapping" WHERE "topicId" = $1`
	if excludeRejected {
		query += ` AND status != 'REJECTED'`
	}

	rows, err := r.pool.Query(ctx, query, topicID)
	if err != nil {
		return nil, fmt.Errorf("querying document ids by topic: %w", err)
	}
	defer rows.Close()

	var docIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning document id: %w", err)
		}
		docIDs = append(docIDs, id)
	}

	return docIDs, rows.Err()
}

// FindByID retrieves a mapping by ID, verifying that the document belongs to the user.
func (r *pgxTopicMappingRepository) FindByID(ctx context.Context, id, userID string) (*model.ResourceTopicMapping, error) {
	query := `
		SELECT m.id, m."documentId", m."topicId", m."subtopicId", m."startPage", m."endPage", m.confidence, m.status
		FROM "ResourceTopicMapping" m
		JOIN "Document" d ON m."documentId" = d.id
		WHERE m.id = $1 AND d."userId" = $2
	`

	m := &model.ResourceTopicMapping{}
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&m.ID,
		&m.DocumentID,
		&m.TopicID,
		&m.SubtopicID,
		&m.StartPage,
		&m.EndPage,
		&m.Confidence,
		&m.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying mapping by id: %w", err)
	}

	return m, nil
}

// UpdateStatus updates the review status of a mapping.
func (r *pgxTopicMappingRepository) UpdateStatus(ctx context.Context, id, status string) (*model.ResourceTopicMapping, error) {
	query := `
		UPDATE "ResourceTopicMapping"
		SET status = $1, "updatedAt" = $2
		WHERE id = $3
		RETURNING id, "documentId", "topicId", "subtopicId", "startPage", "endPage", confidence, status
	`

	m := &model.ResourceTopicMapping{}
	err := r.pool.QueryRow(ctx, query, status, time.Now().UTC(), id).Scan(
		&m.ID,
		&m.DocumentID,
		&m.TopicID,
		&m.SubtopicID,
		&m.StartPage,
		&m.EndPage,
		&m.Confidence,
		&m.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("updating mapping status: %w", err)
	}

	return m, nil
}

