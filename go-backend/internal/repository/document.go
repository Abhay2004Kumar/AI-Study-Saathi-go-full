// Package repository implements data access for Documents.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxDocumentRepository struct {
	pool *pgxpool.Pool
}

// NewDocumentRepository creates a new DocumentRepository backed by pgx.
func NewDocumentRepository(pool *pgxpool.Pool) DocumentRepository {
	return &pgxDocumentRepository{pool: pool}
}

// Create inserts a new document record.
func (r *pgxDocumentRepository) Create(ctx context.Context, doc *model.Document) error {
	now := time.Now().UTC()
	doc.CreatedAt = now
	doc.UpdatedAt = now

	err := r.pool.QueryRow(ctx,
		`INSERT INTO "Document" (
			id, "userId", "examId", title, "fileName", "filePath", 
			"fileType", "fileSize", "fileHash", category, "processingStatus",
			"createdAt", "updatedAt"
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING "createdAt", "updatedAt"`,
		doc.ID,
		doc.UserID,
		doc.ExamID,
		doc.Title,
		doc.FileName,
		doc.FilePath,
		doc.FileType,
		doc.FileSize,
		doc.FileHash,
		doc.Category,
		doc.ProcessingStatus,
		now,
		now,
	).Scan(&doc.CreatedAt, &doc.UpdatedAt)

	if err != nil {
		return fmt.Errorf("inserting document: %w", err)
	}
	return nil
}

// FindByID retrieves a document by ID.
func (r *pgxDocumentRepository) FindByID(ctx context.Context, id string) (*model.Document, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT 
			id, "userId", "examId", title, "fileName", "filePath", 
			"fileType", "fileSize", "fileHash", "pageCount", category, 
			"processingStatus", "aiCategory", "classificationConfidence", 
			"createdAt", "updatedAt"
		 FROM "Document"
		 WHERE id = $1`,
		id,
	)

	doc := &model.Document{}
	err := row.Scan(
		&doc.ID,
		&doc.UserID,
		&doc.ExamID,
		&doc.Title,
		&doc.FileName,
		&doc.FilePath,
		&doc.FileType,
		&doc.FileSize,
		&doc.FileHash,
		&doc.PageCount,
		&doc.Category,
		&doc.ProcessingStatus,
		&doc.AICategory,
		&doc.ClassificationConfidence,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying document by id: %w", err)
	}

	return doc, nil
}

// FindAllByUser returns documents for a user, optionally filtered by examId.
func (r *pgxDocumentRepository) FindAllByUser(ctx context.Context, userID string, examID *string) ([]model.Document, error) {
	query := `
		SELECT 
			id, "userId", "examId", title, "fileName", "filePath", 
			"fileType", "fileSize", "fileHash", "pageCount", category, 
			"processingStatus", "aiCategory", "classificationConfidence", 
			"createdAt", "updatedAt"
		FROM "Document"
		WHERE "userId" = $1
	`
	args := []interface{}{userID}

	if examID != nil && *examID != "" {
		query += ` AND "examId" = $2`
		args = append(args, *examID)
	}

	query += ` ORDER BY "createdAt" DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying documents: %w", err)
	}
	defer rows.Close()

	var documents []model.Document
	for rows.Next() {
		var doc model.Document
		err := rows.Scan(
			&doc.ID,
			&doc.UserID,
			&doc.ExamID,
			&doc.Title,
			&doc.FileName,
			&doc.FilePath,
			&doc.FileType,
			&doc.FileSize,
			&doc.FileHash,
			&doc.PageCount,
			&doc.Category,
			&doc.ProcessingStatus,
			&doc.AICategory,
			&doc.ClassificationConfidence,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning document row: %w", err)
		}
		documents = append(documents, doc)
	}

	return documents, nil
}

// FindByHash finds if a file with this exact SHA-256 hash was already uploaded by this user.
func (r *pgxDocumentRepository) FindByHash(ctx context.Context, userID, hash string) (*model.Document, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, "userId", title, "fileHash"
		 FROM "Document"
		 WHERE "userId" = $1 AND "fileHash" = $2
		 LIMIT 1`,
		userID, hash,
	)

	doc := &model.Document{}
	err := row.Scan(&doc.ID, &doc.UserID, &doc.Title, &doc.FileHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not a duplicate
		}
		return nil, fmt.Errorf("checking file hash: %w", err)
	}

	return doc, nil
}

// UpdateStatus updates the processing status (UPLOADED, PROCESSING, READY, FAILED).
func (r *pgxDocumentRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE "Document" SET "processingStatus" = $1, "updatedAt" = $2 WHERE id = $3`,
		status, time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("updating document status: %w", err)
	}
	return nil
}

// UpdatePageCount sets the page count of a document.
func (r *pgxDocumentRepository) UpdatePageCount(ctx context.Context, id string, count int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE "Document" SET "pageCount" = $1, "updatedAt" = $2 WHERE id = $3`,
		count, time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("updating page count: %w", err)
	}
	return nil
}

// Update performs dynamic column updates on a Document.
func (r *pgxDocumentRepository) Update(ctx context.Context, id string, updates map[string]interface{}) (*model.Document, error) {
	if len(updates) == 0 {
		return r.FindByID(ctx, id)
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for col, val := range updates {
		setClauses = append(setClauses, fmt.Sprintf(`"%s" = $%d`, col, argIdx))
		args = append(args, val)
		argIdx++
	}

	setClauses = append(setClauses, fmt.Sprintf(`"updatedAt" = $%d`, argIdx))
	args = append(args, time.Now().UTC())
	argIdx++

	args = append(args, id)
	query := fmt.Sprintf(`UPDATE "Document" SET %s WHERE id = $%d`, strings.Join(setClauses, ", "), argIdx)

	cmd, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("updating document: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return nil, nil
	}

	return r.FindByID(ctx, id)
}

// Delete removes the document from the database.
func (r *pgxDocumentRepository) Delete(ctx context.Context, id string) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM "Document" WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting document: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

