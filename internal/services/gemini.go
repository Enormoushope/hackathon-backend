package services

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/vertexai/genai"
)

func GetGeminiClient(ctx context.Context) (*genai.Client, error) {
	// ここに citric-earth-477705-r6 が入るように環境変数を設定
	projectID := os.Getenv("GCP_PROJECT_ID") 
	location := "us-central1" // 東京なら asia-northeast1

	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// 商品説明の自動生成
func GenerateDescription(title string) (string, error) {
	ctx := context.Background()
	client, err := GetGeminiClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash-002")
	
	prompt := genai.Text(fmt.Sprintf("フリマアプリで「%s」を出品します。魅力的で詳細な商品説明文を日本語で生成してください。", title))
	resp, err := model.GenerateContent(ctx, prompt)
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
	}
	return "説明文を生成できませんでした", nil
}

func SuggestPrice(title, description string) (string, error) {
	ctx := context.Background()
	client, err := GetGeminiClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash-002")

	// 査定用のプロンプト
	prompt := genai.Text(fmt.Sprintf(
		"商品名:%s, 説明文:%s。この商品のフリマアプリでの中古市場価格（日本円）を査定し、理由を添えて金額のみを太字で、それ以外を簡潔に答えてください。", 
		title, description,
	))

	resp, err := model.GenerateContent(ctx, prompt)
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
	}
	return "価格を査定できませんでした", nil
}