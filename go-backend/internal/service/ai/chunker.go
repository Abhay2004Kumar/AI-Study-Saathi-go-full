// Package ai provides text chunking, embeddings, and RAG pipelines.
package ai

import (
	"strings"
)

// TextChunk represents an extracted, indexed segment of text from a document.
type TextChunk struct {
	Index   int
	Content string
	Page    *int
}

// Chunker splits large text into overlapping chunks respecting natural boundaries.
type Chunker struct {
	ChunkSize    int
	ChunkOverlap int
	Separators   []string
}

// NewDefaultChunker creates a chunker matching LangChain defaults (1000 size, 200 overlap).
func NewDefaultChunker() *Chunker {
	return &Chunker{
		ChunkSize:    1000,
		ChunkOverlap: 200,
		Separators:   []string{"\n\n", "\n", ". ", " ", ""},
	}
}

// SplitText splits a raw text string into chunks.
func (c *Chunker) SplitText(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}
	if len(text) <= c.ChunkSize {
		return []string{text}
	}

	return c.recursiveSplit(text, c.Separators)
}

func (c *Chunker) recursiveSplit(text string, separators []string) []string {
	var finalChunks []string
	separator := ""
	var nextSeparators []string

	for i, sep := range separators {
		if sep == "" || strings.Contains(text, sep) {
			separator = sep
			nextSeparators = separators[i+1:]
			break
		}
	}

	var splits []string
	if separator == "" {
		// Split by characters if no separator matched
		for _, r := range text {
			splits = append(splits, string(r))
		}
	} else {
		splits = strings.Split(text, separator)
	}

	var currentDoc []string
	totalLen := 0

	for _, piece := range splits {
		pieceLen := len(piece)
		if totalLen+pieceLen+len(separator) > c.ChunkSize && len(currentDoc) > 0 {
			doc := strings.Join(currentDoc, separator)
			doc = strings.TrimSpace(doc)
			if doc != "" {
				if len(doc) > c.ChunkSize && len(nextSeparators) > 0 {
					subChunks := c.recursiveSplit(doc, nextSeparators)
					finalChunks = append(finalChunks, subChunks...)
				} else {
					finalChunks = append(finalChunks, doc)
				}
			}

			// Keep overlap from end of currentDoc
			for len(currentDoc) > 0 && totalLen > c.ChunkOverlap {
				totalLen -= len(currentDoc[0]) + len(separator)
				currentDoc = currentDoc[1:]
			}
		}

		currentDoc = append(currentDoc, piece)
		totalLen += pieceLen + len(separator)
	}

	if len(currentDoc) > 0 {
		doc := strings.TrimSpace(strings.Join(currentDoc, separator))
		if doc != "" {
			if len(doc) > c.ChunkSize && len(nextSeparators) > 0 {
				subChunks := c.recursiveSplit(doc, nextSeparators)
				finalChunks = append(finalChunks, subChunks...)
			} else {
				finalChunks = append(finalChunks, doc)
			}
		}
	}

	return finalChunks
}

