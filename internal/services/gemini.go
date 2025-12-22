package services

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/vertexai/genai"
)

func GetGeminiClient(ctx context.Context) (*genai.Client, error) {
	// os.Getenv("GCP_PROJECT_ID") が本当に取れているかチェック
	projectID := os.Getenv("GCP_PROJECT_ID")
	location := ""

	// 🔴 もし環境変数が空なら、エラーメッセージにそれを混ぜる
	if projectID == "" {
		return nil, fmt.Errorf("GCP_PROJECT_ID is empty. Please check Cloud Run env settings")
	}

	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		return nil, fmt.Errorf("genai.NewClient creation failed: %w", err)
	}
	return client, nil
}

// 商品説明の自動生成
func GenerateDescription(title string, base64Data string) (string, error) {
	ctx := context.Background()
	client, err := GetGeminiClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.0-flash-exp")

	// --- 画像データの処理 ---
	var prompt []genai.Part
	
	if base64Data != "" {
		// "data:image/jpeg;base64," などのヘッダーを除去
		parts := strings.Split(base64Data, ",")
		rawBase64 := parts[len(parts)-1]
		
		data, err := base64.StdEncoding.DecodeString(rawBase64)
		if err == nil {
			// 画像をプロンプトに含める
			prompt = append(prompt, genai.ImageData("jpeg", data))
		}
	}

	// テキストを追加
	promptText := fmt.Sprintf("商品名「%s」とこの画像を見て、魅力的な商品説明を100文字程度で作成してください。", title)
	prompt = append(prompt, genai.Text(promptText))

	resp, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		return "", fmt.Errorf("Gemini生成エラー: %w", err)
	}

	if len(resp.Candidates) > 0 {
		return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
	}
	return "", fmt.Errorf("AIからの回答が空でした")
}

func SuggestPrice(title, description string) (string, error) {
	ctx := context.Background()
	client, err := GetGeminiClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.0-flash")

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