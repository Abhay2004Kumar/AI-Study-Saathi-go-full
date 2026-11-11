// Package gemini provides a clean, production-grade HTTP client for the Google Gemini API.
package gemini

// Content represents a chat message or content block in Gemini API.
type Content struct {
	Role  string `json:"role,omitempty"` // "user", "model"
	Parts []Part `json:"parts"`
}

// InlineData represents base64-encoded media data for multimodal requests.
type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64-encoded string
}

// Part represents a part of a Content block.
type Part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *InlineData `json:"inlineData,omitempty"`
}

// GenerationConfig configures model output parameters.
type GenerationConfig struct {
	Temperature      float64 `json:"temperature,omitempty"`
	TopP             float64 `json:"topP,omitempty"`
	TopK             int     `json:"topK,omitempty"`
	MaxOutputTokens  int     `json:"maxOutputTokens,omitempty"`
	ResponseMimeType string  `json:"responseMimeType,omitempty"` // "application/json" or "text/plain"
}

// SystemInstruction provides system-level guidance to the model.
type SystemInstruction struct {
	Parts []Part `json:"parts"`
}

// GenerateContentRequest is the payload sent to :generateContent
type GenerateContentRequest struct {
	Contents          []Content          `json:"contents"`
	GenerationConfig  *GenerationConfig  `json:"generationConfig,omitempty"`
	SystemInstruction *SystemInstruction `json:"systemInstruction,omitempty"`
}

// Candidate represents a generated response option.
type Candidate struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason"`
}

// GenerateContentResponse is the payload received from :generateContent
type GenerateContentResponse struct {
	Candidates []Candidate `json:"candidates"`
	Error      *APIError   `json:"error,omitempty"`
}

// EmbedContentRequest is the payload sent to :embedContent
type EmbedContentRequest struct {
	Model   string  `json:"model"`
	Content Content `json:"content"`
}

// ContentEmbedding represents a single vector embedding.
type ContentEmbedding struct {
	Values []float32 `json:"values"`
}

// EmbedContentResponse is the payload received from :embedContent
type EmbedContentResponse struct {
	Embedding *ContentEmbedding `json:"embedding,omitempty"`
	Error     *APIError         `json:"error,omitempty"`
}

// BatchEmbedContentsRequest is the payload sent to :batchEmbedContents
type BatchEmbedContentsRequest struct {
	Requests []EmbedContentRequest `json:"requests"`
}

// BatchEmbedContentsResponse is the payload received from :batchEmbedContents
type BatchEmbedContentsResponse struct {
	Embeddings []ContentEmbedding `json:"embeddings"`
	Error      *APIError          `json:"error,omitempty"`
}

// APIError represents an error returned by the Gemini REST API.
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

