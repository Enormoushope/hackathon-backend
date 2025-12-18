package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test_vertexai.go <prompt>")
		fmt.Println("Example: go run test_vertexai.go \"日本の首都は？\"")
		os.Exit(1)
	}

	prompt := os.Args[1]
	err := generateContentFromText(prompt)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func generateContentFromText(promptText string) error {
	ctx := context.Background()

	projectID := getEnvOrDefault("GOOGLE_CLOUD_PROJECT_ID", "your-project-id")
	location := getEnvOrDefault("GOOGLE_CLOUD_LOCATION", "us-central1")
	modelName := getEnvOrDefault("GOOGLE_MODEL_NAME", "gemini-2.0-flash-001")

	// Pythonコードと同じくAPI key + VertexAI backend を使用
	apiKey := os.Getenv("GOOGLE_CLOUD_API_KEY")
	if apiKey == "" {
		// API keyがなければADCを使う
		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			Project:  projectID,
			Location: location,
			Backend:  genai.BackendVertexAI,
		})
		if err != nil {
			return fmt.Errorf("error creating client with ADC: %w", err)
		}

		resp, err := client.Models.GenerateContent(ctx, modelName, genai.Text(promptText), nil)
		if err != nil {
			return fmt.Errorf("error generating content: %w", err)
		}

		for _, cand := range resp.Candidates {
			for _, part := range cand.Content.Parts {
				fmt.Print(part)
			}
		}
		fmt.Println()
		return nil
	}

	// API keyを使用
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendVertexAI,
	})
	if err != nil {
		return fmt.Errorf("error creating client with API key: %w", err)
	}

	resp, err := client.Models.GenerateContent(ctx, modelName, genai.Text(promptText), nil)
	if err != nil {
		return fmt.Errorf("error generating content: %w", err)
	}

	for _, cand := range resp.Candidates {
		for _, part := range cand.Content.Parts {
			fmt.Print(part)
		}
	}
	fmt.Println()
	return nil
}
