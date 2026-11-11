package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL        = "https://generativelanguage.googleapis.com/v1beta"
	defaultGenerateModel  = "gemini-3.5-flash-lite"
	defaultEmbeddingModel = "gemini-embedding-2"
)

// Client is a goroutine-safe HTTP client for the Google Gemini API.
type Client struct {
	apiKey         string
	baseURL        string
	generateModel  string
	embeddingModel string
	httpClient     *http.Client
	retryConfig    RetryConfig
}

// NewClient creates a configured Gemini API client.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:         apiKey,
		baseURL:        defaultBaseURL,
		generateModel:  defaultGenerateModel,
		embeddingModel: defaultEmbeddingModel,
		retryConfig:    DefaultRetryConfig(),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// ClientOption allows customizing Client properties.
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL (useful for testing with httptest).
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithGenerateModel sets the default model used for text generation.
func WithGenerateModel(model string) ClientOption {
	return func(c *Client) {
		c.generateModel = model
	}
}

// WithEmbeddingModel sets the default model used for vector embeddings.
func WithEmbeddingModel(model string) ClientOption {
	return func(c *Client) {
		c.embeddingModel = model
	}
}

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithRetryConfig customizes backoff parameters.
func WithRetryConfig(cfg RetryConfig) ClientOption {
	return func(c *Client) {
		c.retryConfig = cfg
	}
}

// GenerateContent generates text from a prompt, with optional system prompt and temperature.
func (c *Client) GenerateContent(ctx context.Context, prompt string, systemInstruction string, temp float64) (string, error) {
	reqBody := GenerateContentRequest{
		Contents: []Content{
			{
				Role:  "user",
				Parts: []Part{{Text: prompt}},
			},
		},
		GenerationConfig: &GenerationConfig{
			Temperature: temp,
		},
	}

	if systemInstruction != "" {
		reqBody.SystemInstruction = &SystemInstruction{
			Parts: []Part{{Text: systemInstruction}},
		}
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.generateModel, c.apiKey)

	resp, err := WithRetry(ctx, c.retryConfig, func() (*GenerateContentResponse, int, string, error) {
		target := &GenerateContentResponse{}
		status, body, err := c.doPost(ctx, endpoint, reqBody, target)
		return target, status, body, err
	})
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no content candidates")
	}

	return resp.Candidates[0].Content.Parts[0].Text, nil
}

// ExtractTextFromImage transcribes legible text from an image using Gemini Vision.
func (c *Client) ExtractTextFromImage(ctx context.Context, mimeType string, imageBytes []byte) (string, error) {
	if len(imageBytes) == 0 {
		return "", nil
	}

	b64Data := base64.StdEncoding.EncodeToString(imageBytes)
	prompt := `Transcribe every piece of legible text in this image exactly as written — handwritten or printed notes, headings, bullet points, questions, diagram labels, formulas, anything readable. Preserve structure (headings, lists) where clear, but write in plain text only: no LaTeX (no $...$, \rightarrow, \ge, etc.) and no markdown (no **bold**, no backticks) — write formulas and notation in plain words or ordinary keyboard symbols instead, e.g. "X -> A". If nothing in the image is legible, write NO_TEXT.`

	reqBody := GenerateContentRequest{
		Contents: []Content{
			{
				Role: "user",
				Parts: []Part{
					{Text: prompt},
					{
						InlineData: &InlineData{
							MimeType: mimeType,
							Data:     b64Data,
						},
					},
				},
			},
		},
		GenerationConfig: &GenerationConfig{
			Temperature: 0,
		},
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.generateModel, c.apiKey)

	resp, err := WithRetry(ctx, c.retryConfig, func() (*GenerateContentResponse, int, string, error) {
		target := &GenerateContentResponse{}
		status, body, err := c.doPost(ctx, endpoint, reqBody, target)
		return target, status, body, err
	})
	if err != nil {
		return "", fmt.Errorf("gemini vision transcription: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", nil
	}

	extracted := strings.TrimSpace(resp.Candidates[0].Content.Parts[0].Text)
	if extracted == "NO_TEXT" || strings.EqualFold(extracted, "no readable text") || strings.EqualFold(extracted, "no text") {
		return "", nil
	}

	return extracted, nil
}

// GenerateJSON generates structured output, automatically instructing Gemini to output valid JSON
// and unmarshaling it directly into destination target.
func (c *Client) GenerateJSON(ctx context.Context, prompt string, systemInstruction string, temp float64, target interface{}) error {
	reqBody := GenerateContentRequest{
		Contents: []Content{
			{
				Role:  "user",
				Parts: []Part{{Text: prompt}},
			},
		},
		GenerationConfig: &GenerationConfig{
			Temperature:      temp,
			ResponseMimeType: "application/json",
		},
	}

	if systemInstruction != "" {
		reqBody.SystemInstruction = &SystemInstruction{
			Parts: []Part{{Text: systemInstruction}},
		}
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.generateModel, c.apiKey)

	resp, err := WithRetry(ctx, c.retryConfig, func() (*GenerateContentResponse, int, string, error) {
		target := &GenerateContentResponse{}
		status, body, err := c.doPost(ctx, endpoint, reqBody, target)
		return target, status, body, err
	})
	if err != nil {
		return err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return fmt.Errorf("gemini returned no content candidates")
	}

	rawText := strings.TrimSpace(resp.Candidates[0].Content.Parts[0].Text)
	// Clean markdown code blocks if present (```json ... ```)
	rawText = strings.TrimPrefix(rawText, "```json")
	rawText = strings.TrimPrefix(rawText, "```")
	rawText = strings.TrimSuffix(rawText, "```")
	rawText = strings.TrimSpace(rawText)

	if err := json.Unmarshal([]byte(rawText), target); err != nil {
		return fmt.Errorf("parsing structured JSON output: %w (raw response: %s)", err, rawText)
	}

	return nil
}

// EmbedContent generates a single 3072-dimensional vector embedding for a piece of text.
func (c *Client) EmbedContent(ctx context.Context, text string) ([]float32, error) {
	reqBody := EmbedContentRequest{
		Model: "models/" + c.embeddingModel,
		Content: Content{
			Parts: []Part{{Text: text}},
		},
	}

	endpoint := fmt.Sprintf("%s/models/%s:embedContent?key=%s", c.baseURL, c.embeddingModel, c.apiKey)

	resp, err := WithRetry(ctx, c.retryConfig, func() (*EmbedContentResponse, int, string, error) {
		target := &EmbedContentResponse{}
		status, body, err := c.doPost(ctx, endpoint, reqBody, target)
		return target, status, body, err
	})
	if err != nil {
		return nil, err
	}

	if resp.Embedding == nil || len(resp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("no embedding values returned by gemini")
	}

	return resp.Embedding.Values, nil
}

// BatchEmbedContents generates vector embeddings for multiple texts, safely chunking requests
// to respect Gemini's max 100 texts per batchEmbedContents limit.
func (c *Client) BatchEmbedContents(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	const maxBatchSize = 100
	results := make([][]float32, 0, len(texts))

	for i := 0; i < len(texts); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(texts) {
			end = len(texts)
		}
		batchSlice := texts[i:end]

		requests := make([]EmbedContentRequest, len(batchSlice))
		for j, text := range batchSlice {
			requests[j] = EmbedContentRequest{
				Model: "models/" + c.embeddingModel,
				Content: Content{
					Parts: []Part{{Text: text}},
				},
			}
		}

		reqBody := BatchEmbedContentsRequest{Requests: requests}
		endpoint := fmt.Sprintf("%s/models/%s:batchEmbedContents?key=%s", c.baseURL, c.embeddingModel, c.apiKey)

		resp, err := WithRetry(ctx, c.retryConfig, func() (*BatchEmbedContentsResponse, int, string, error) {
			target := &BatchEmbedContentsResponse{}
			status, body, err := c.doPost(ctx, endpoint, reqBody, target)
			return target, status, body, err
		})
		if err != nil {
			return nil, err
		}

		for _, emb := range resp.Embeddings {
			results = append(results, emb.Values)
		}
	}

	return results, nil
}

// Helper: JSON POST request executor
func (c *Client) doPost(ctx context.Context, url string, reqBody interface{}, target interface{}) (int, string, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, "", fmt.Errorf("marshaling request payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, "", fmt.Errorf("creating http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, "", err
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return httpResp.StatusCode, "", fmt.Errorf("reading response body: %w", err)
	}

	bodyStr := string(respBytes)

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return httpResp.StatusCode, bodyStr, fmt.Errorf("http error status %d: %s", httpResp.StatusCode, strings.TrimSpace(bodyStr))
	}

	if err := json.Unmarshal(respBytes, target); err != nil {
		return httpResp.StatusCode, bodyStr, fmt.Errorf("decoding response json: %w", err)
	}

	return httpResp.StatusCode, bodyStr, nil
}
