// Package worker implements document ingestion (chunking, embedding, storage).
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/internal/service/ai"
	"github.com/aiStudySaathi/backend/pkg/gemini"
)

// DocumentProcessor orchestrates parsing, chunking, and embedding for uploaded documents.
type DocumentProcessor struct {
	docRepo        repository.DocumentRepository
	chunkRepo      repository.DocumentChunkRepository
	geminiClient   *gemini.Client
	chunker        *ai.Chunker
	mappingService service.TopicMappingService
	pyqService     service.PyqService
}

// NewDocumentProcessor creates a new DocumentProcessor.
func NewDocumentProcessor(
	docRepo repository.DocumentRepository,
	chunkRepo repository.DocumentChunkRepository,
	geminiClient *gemini.Client,
	mappingService service.TopicMappingService,
	pyqService service.PyqService,
) *DocumentProcessor {
	return &DocumentProcessor{
		docRepo:        docRepo,
		chunkRepo:      chunkRepo,
		geminiClient:   geminiClient,
		chunker:        ai.NewDefaultChunker(),
		mappingService: mappingService,
		pyqService:     pyqService,
	}
}

// Process processes a single document by ID.
func (dp *DocumentProcessor) Process(ctx context.Context, documentID string) error {
	// 1. Fetch document from DB
	doc, err := dp.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("fetching document %s: %w", documentID, err)
	}
	if doc == nil {
		slog.Warn("Document deleted before processing could start; skipping", "documentId", documentID)
		return nil
	}

	// 2. Mark status as PROCESSING
	_ = dp.docRepo.UpdateStatus(ctx, documentID, "PROCESSING")

	// 3. Extract text from document (PDF, TXT, MD, Images) with UTF-8 sanitization
	rawText, err := ai.ExtractTextFromFile(ctx, doc.FilePath, dp.geminiClient)
	if err != nil {
		slog.Error("Failed to extract text from document", "documentId", documentID, "error", err)
		_ = dp.docRepo.UpdateStatus(ctx, documentID, "FAILED")
		return fmt.Errorf("extracting text from file: %w", err)
	}

	// If empty file or no readable text, mark READY with 0 chunks
	if len(rawText) == 0 {
		slog.Warn("Document contains no readable text", "documentId", documentID)
		_ = dp.docRepo.UpdateStatus(ctx, documentID, "READY")
		return nil
	}

	// 4. Split into chunks
	textChunks := dp.chunker.SplitText(rawText)
	if len(textChunks) == 0 {
		_ = dp.docRepo.UpdateStatus(ctx, documentID, "READY")
		return nil
	}

	// 5. Generate embeddings in batch via Gemini
	embeddings, err := dp.geminiClient.BatchEmbedContents(ctx, textChunks)
	if err != nil {
		slog.Error("Failed to generate embeddings, saving chunks without vectors", "error", err)
		embeddings = nil
	}

	// 6. Delete any existing chunks for this document (retry scenario)
	_ = dp.chunkRepo.DeleteByDocumentID(ctx, documentID)

	// 7. Insert Chunks into pgvector
	for i, chunkText := range textChunks {
		cleanChunk := strings.ToValidUTF8(chunkText, "")
		cleanChunk = strings.ReplaceAll(cleanChunk, "\x00", "")
		chunk := &model.DocumentChunk{
			DocumentID: documentID,
			Content:    cleanChunk,
			ChunkIndex: i,
			Metadata:   []byte("{}"),
		}

		if embeddings != nil && i < len(embeddings) && len(embeddings[i]) > 0 {
			if err := dp.chunkRepo.InsertWithEmbedding(ctx, chunk, embeddings[i]); err != nil {
				slog.Error("Error inserting chunk with vector", "index", i, "error", err)
				_ = dp.chunkRepo.InsertWithoutEmbedding(ctx, chunk)
			}
		} else {
			_ = dp.chunkRepo.InsertWithoutEmbedding(ctx, chunk)
		}
	}

	// 8. If linked to an exam, run Topic Mapping (best-effort)
	if dp.mappingService != nil && doc.ExamID != nil && *doc.ExamID != "" {
		if err := dp.mappingService.Run(ctx, documentID); err != nil {
			slog.Warn("Topic mapping failed for document, continuing", "documentId", documentID, "error", err)
		}
	}

	// 9. If PYQ category, run PYQ Analysis (best-effort)
	isPyq := doc.Category == "PYQ" || (doc.AICategory != nil && *doc.AICategory == "PYQ")
	if dp.pyqService != nil && isPyq && doc.ExamID != nil && *doc.ExamID != "" {
		if _, err := dp.pyqService.AnalyzePyqDocument(ctx, doc.UserID, documentID); err != nil {
			slog.Warn("PYQ analysis failed for document, continuing", "documentId", documentID, "error", err)
		}
	}

	// 10. Mark status as READY
	_ = dp.docRepo.UpdateStatus(ctx, documentID, "READY")
	slog.Info("Document successfully processed and vectorized", "documentId", documentID, "chunkCount", len(textChunks))

	return nil
}
