// Package ai implements the RAG question-answering pipeline.
package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/gemini"
)

// RAGService defines the RAG Q&A interface.
type RAGService interface {
	AskQuestion(ctx context.Context, userID, question string, filters RAGFilters) (*RAGResponse, error)
}

// RAGFilters specifies retrieval narrowing.
type RAGFilters struct {
	ExamID   *string
	Category *string
}

// RAGSource represents a cited document.
type RAGSource struct {
	Title string `json:"title"`
	Page  *int   `json:"page"`
}

// RAGResponse matches the contract expected by the frontend.
type RAGResponse struct {
	Answer           string      `json:"answer"`
	FoundInMaterials bool        `json:"foundInMaterials"`
	Sources          []RAGSource `json:"sources"`
}

type ragService struct {
	chunkRepo    repository.DocumentChunkRepository
	geminiClient *gemini.Client
}

// NewRAGService creates a new RAGService.
func NewRAGService(chunkRepo repository.DocumentChunkRepository, geminiClient *gemini.Client) RAGService {
	return &ragService{
		chunkRepo:    chunkRepo,
		geminiClient: geminiClient,
	}
}

// LLMRAGOutput defines the expected JSON shape from Gemini.
type LLMRAGOutput struct {
	Answer           string   `json:"answer"`
	FoundInMaterials bool     `json:"foundInMaterials"`
	UsedSourceIDs    []string `json:"usedSourceIds"`
}

func (s *ragService) AskQuestion(ctx context.Context, userID, question string, filters RAGFilters) (*RAGResponse, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, apperror.BadRequest("Question cannot be empty")
	}

	// 1. Generate Embedding for the user question
	queryVector, err := s.geminiClient.EmbedContent(ctx, question)
	if err != nil {
		return nil, fmt.Errorf("generating query embedding: %w", err)
	}

	// 2. Query pgvector for closest matching chunks
	chunks, err := s.chunkRepo.SimilaritySearch(ctx, repository.SimilaritySearchParams{
		UserID:      userID,
		QueryVector: queryVector,
		TopK:        5,
		ExamID:      filters.ExamID,
		Category:    filters.Category,
	})
	if err != nil {
		return nil, fmt.Errorf("retrieving chunks from pgvector: %w", err)
	}

	// 3. Format context text
	var contextBuilder strings.Builder
	if len(chunks) == 0 {
		contextBuilder.WriteString("No sources were found in the student's materials.")
	} else {
		for i, ch := range chunks {
			contextBuilder.WriteString(fmt.Sprintf("[S%d] %s:\n%s\n\n", i+1, ch.Title, ch.Content))
		}
	}

	// 4. Construct prompt instructions matching rag.prompt.js
	prompt := fmt.Sprintf(`
NUMBERED SOURCES FROM THE STUDENT'S OWN UPLOADED MATERIALS:
%s

QUESTION:
%s

INSTRUCTIONS:
- Answer using the numbered sources above whenever they actually contain the answer.
- In usedSourceIds, list only the labels (e.g. "S1", "S2") of sources you truly drew from. Never list a source you didn't use.
- If the sources don't contain the answer, set foundInMaterials to false and usedSourceIds to an empty array. You may still answer from general knowledge, but you MUST clearly say the answer isn't from their uploaded materials (e.g. start with "I couldn't find this in your uploaded materials, but...").
- The answer is displayed as plain text with no math or markdown rendering. NEVER use LaTeX syntax and never wrap terms in markdown (no bold, no backticks). Write formulas in plain words or unicode symbols.
`, contextBuilder.String(), question)

	systemInstruction := "You are a helpful AI study companion answering a student's question based strictly on their own course materials."

	// 5. Invoke Gemini with structured JSON output
	var llmOutput LLMRAGOutput
	err = s.geminiClient.GenerateJSON(ctx, prompt, systemInstruction, 0.2, &llmOutput)
	if err != nil {
		return nil, fmt.Errorf("gemini generation failed: %w", err)
	}

	// 6. Defensively resolve citations (strip any hallucinated source IDs!)
	resolvedSources := s.resolveSources(chunks, llmOutput)

	return &RAGResponse{
		Answer:           llmOutput.Answer,
		FoundInMaterials: llmOutput.FoundInMaterials && len(chunks) > 0,
		Sources:          resolvedSources,
	}, nil
}

func (s *ragService) resolveSources(chunks []repository.SimilarityResult, output LLMRAGOutput) []RAGSource {
	if len(chunks) == 0 || !output.FoundInMaterials {
		return []RAGSource{}
	}

	usedMap := make(map[string]bool)
	for _, id := range output.UsedSourceIDs {
		usedMap[strings.ToUpper(strings.TrimSpace(id))] = true
	}

	seen := make(map[string]bool)
	var sources []RAGSource

	for i, ch := range chunks {
		label := fmt.Sprintf("S%d", i+1)
		if !usedMap[label] {
			continue
		}

		key := ch.Title
		if seen[key] {
			continue
		}
		seen[key] = true

		sources = append(sources, RAGSource{
			Title: ch.Title,
			Page:  nil, // Can be parsed from metadata if present
		})
	}

	return sources
}

