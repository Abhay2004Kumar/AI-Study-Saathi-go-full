// Package repository implements data access for DocumentChunks and pgvector similarity queries.
package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxDocumentChunkRepository struct {
	pool *pgxpool.Pool
}

// NewDocumentChunkRepository creates a new DocumentChunkRepository backed by pgx.
func NewDocumentChunkRepository(pool *pgxpool.Pool) DocumentChunkRepository {
	return &pgxDocumentChunkRepository{pool: pool}
}

// formatVector formats []float32 into a pgvector string "[0.123,-0.456,...]"
func formatVector(vec []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(fmt.Sprintf("%f", v))
	}
	sb.WriteByte(']')
	return sb.String()
}

// InsertWithEmbedding inserts a DocumentChunk with its pgvector embedding literal.
func (r *pgxDocumentChunkRepository) InsertWithEmbedding(ctx context.Context, chunk *model.DocumentChunk, embedding []float32) error {
	id := uuid.New().String()
	vectorStr := formatVector(embedding)

	query := `
		INSERT INTO "DocumentChunk" ("id", "documentId", "content", "chunkIndex", "metadata", "embedding")
		VALUES ($1, $2, $3, $4, $5, $6::vector)
	`
	_, err := r.pool.Exec(ctx, query, id, chunk.DocumentID, chunk.Content, chunk.ChunkIndex, chunk.Metadata, vectorStr)
	if err != nil {
		return fmt.Errorf("inserting chunk with vector: %w", err)
	}
	return nil
}

// InsertWithoutEmbedding inserts a chunk without an embedding (fallback).
func (r *pgxDocumentChunkRepository) InsertWithoutEmbedding(ctx context.Context, chunk *model.DocumentChunk) error {
	id := uuid.New().String()
	query := `
		INSERT INTO "DocumentChunk" ("id", "documentId", "content", "chunkIndex", "metadata")
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query, id, chunk.DocumentID, chunk.Content, chunk.ChunkIndex, chunk.Metadata)
	if err != nil {
		return fmt.Errorf("inserting chunk without vector: %w", err)
	}
	return nil
}

// SimilaritySearch performs cosine distance search (<=> operator) using pgvector.
func (r *pgxDocumentChunkRepository) SimilaritySearch(ctx context.Context, params SimilaritySearchParams) ([]SimilarityResult, error) {
	if len(params.QueryVector) == 0 {
		return nil, fmt.Errorf("query vector cannot be empty")
	}

	vectorStr := formatVector(params.QueryVector)
	topK := params.TopK
	if topK <= 0 {
		topK = 5
	}

	query := `
		SELECT 
			c.content, 
			c.metadata, 
			d.id AS "documentId", 
			d.title, 
			d."examId", 
			d.category, 
			(c."embedding" <=> $1::vector) AS distance
		FROM "DocumentChunk" c
		JOIN "Document" d ON c."documentId" = d.id
		WHERE d."userId" = $2
	`
	args := []interface{}{vectorStr, params.UserID}
	argIdx := 3

	if params.ExamID != nil && *params.ExamID != "" {
		query += fmt.Sprintf(` AND d."examId" = $%d`, argIdx)
		args = append(args, *params.ExamID)
		argIdx++
	}

	if params.Category != nil && *params.Category != "" {
		query += fmt.Sprintf(` AND d.category = $%d`, argIdx)
		args = append(args, *params.Category)
		argIdx++
	}

	if len(params.DocumentIDs) > 0 {
		query += fmt.Sprintf(` AND d.id = ANY($%d)`, argIdx)
		args = append(args, params.DocumentIDs)
		argIdx++
	}

	query += fmt.Sprintf(` ORDER BY c."embedding" <=> $1::vector LIMIT $%d`, argIdx)
	args = append(args, topK)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("pgvector similarity search failed: %w", err)
	}
	defer rows.Close()

	var results []SimilarityResult
	for rows.Next() {
		var res SimilarityResult
		err := rows.Scan(
			&res.Content,
			&res.Metadata,
			&res.DocumentID,
			&res.Title,
			&res.ExamID,
			&res.Category,
			&res.Distance,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning similarity row: %w", err)
		}
		results = append(results, res)
	}

	return results, nil
}

// FindByDocumentID returns all chunks for a document ordered by index.
func (r *pgxDocumentChunkRepository) FindByDocumentID(ctx context.Context, documentID string) ([]model.DocumentChunk, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, "documentId", content, "chunkIndex", metadata, "createdAt"
		 FROM "DocumentChunk"
		 WHERE "documentId" = $1
		 ORDER BY "chunkIndex" ASC`,
		documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching chunks by document id: %w", err)
	}
	defer rows.Close()

	var chunks []model.DocumentChunk
	for rows.Next() {
		var c model.DocumentChunk
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.Content, &c.ChunkIndex, &c.Metadata, &c.CreatedAt); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, nil
}

// DeleteByDocumentID removes all chunks for a document.
func (r *pgxDocumentChunkRepository) DeleteByDocumentID(ctx context.Context, documentID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM "DocumentChunk" WHERE "documentId" = $1`, documentID)
	if err != nil {
		return fmt.Errorf("deleting document chunks: %w", err)
	}
	return nil
}

