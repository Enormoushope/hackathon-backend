package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"google.golang.org/genai"
)

// extractJSON extracts the first JSON object from a string.
func extractJSON(text string) string {
	text = regexp.MustCompile("```json\\s*").ReplaceAllString(text, "")
	text = regexp.MustCompile("```\\s*").ReplaceAllString(text, "")

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return strings.TrimSpace(text[start : end+1])
	}
	return text
}

// downloadImage downloads an image from URL and returns bytes and MIME type.
func downloadImage(url string) ([]byte, string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read body: %w", err)
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	return data, mimeType, nil
}

// VertexAIManager manages the VertexAI client.
type VertexAIManager struct {
	projectID string
	location  string
	apiKey    string
	client    *genai.Client
}

// NewVertexAIManager creates a new VertexAI manager.
func NewVertexAIManager(projectID, location string) *VertexAIManager {
	return &VertexAIManager{
		projectID: projectID,
		location:  location,
		apiKey:    os.Getenv("GOOGLE_CLOUD_API_KEY"),
	}
}

// Initialize initializes the VertexAI client.
func (m *VertexAIManager) Initialize(ctx context.Context) error {
	var client *genai.Client
	var err error

	if m.apiKey != "" {
		client, err = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  m.apiKey,
			Backend: genai.BackendVertexAI,
		})
	} else {
		client, err = genai.NewClient(ctx, &genai.ClientConfig{
			Project:  m.projectID,
			Location: m.location,
			Backend:  genai.BackendVertexAI,
		})
	}

	if err != nil {
		return fmt.Errorf("failed to create VertexAI client: %w", err)
	}

	m.client = client
	return nil
}

// Close closes the VertexAI client.
func (m *VertexAIManager) Close() error {
	return nil
}
