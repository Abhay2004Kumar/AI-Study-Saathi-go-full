package ai

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/aiStudySaathi/backend/pkg/gemini"
	"github.com/ledongthuc/pdf"
)

// IsImageFile checks if a file extension corresponds to an image format.
func IsImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp"
}

// GetImageMimeType returns the MIME type for an image extension.
func GetImageMimeType(filePath string) string {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// ExtractTextFromFile extracts plain text from a given document file (PDF, TXT, MD, Images).
// For images (.jpg, .jpeg, .png, .webp), it runs multimodal OCR via Gemini Vision if a geminiClient is provided.
// It cleans the text to ensure 100% valid UTF-8 and strips null bytes (\x00),
// preventing PostgreSQL SQLSTATE 22021 (invalid byte sequence for encoding "UTF8").
func ExtractTextFromFile(ctx context.Context, filePath string, geminiClient ...*gemini.Client) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	var rawText string
	switch ext {
	case ".pdf":
		text, err := readPDF(filePath)
		if err != nil {
			return "", fmt.Errorf("reading pdf %s: %w", filePath, err)
		}
		rawText = text

	case ".jpg", ".jpeg", ".png", ".webp":
		if len(geminiClient) > 0 && geminiClient[0] != nil {
			imgBytes, err := os.ReadFile(filePath)
			if err != nil {
				return "", fmt.Errorf("reading image file %s: %w", filePath, err)
			}
			mimeType := GetImageMimeType(filePath)
			slog.Info("Running Gemini Vision OCR on uploaded image", "filePath", filePath, "mimeType", mimeType)
			text, err := geminiClient[0].ExtractTextFromImage(ctx, mimeType, imgBytes)
			if err != nil {
				slog.Warn("Failed to extract text from image via OCR", "filePath", filePath, "error", err)
				return "", nil
			}
			rawText = text
		} else {
			// No OCR client provided; image has no direct plain text
			return "", nil
		}

	default:
		// Plain text, markdown, or generic text file
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("reading file %s: %w", filePath, err)
		}
		rawText = string(data)
	}

	// Sanitize string for PostgreSQL UTF-8 compliance
	sanitized := strings.ToValidUTF8(rawText, "")
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	return strings.TrimSpace(sanitized), nil
}

func readPDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	_, err = buf.ReadFrom(b)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

