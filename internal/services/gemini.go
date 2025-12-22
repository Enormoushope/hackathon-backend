package services

import (
	"context"
	"fmt"
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
func GenerateDescription(title string) (string, error) {
	ctx := context.Background()
	client, err := GetGeminiClient(ctx)
	if err != nil {
		return "", fmt.Errorf("【致命的】クライアント作成失敗: %w", err)
	}
	defer client.Close()

	// 試したいモデル名を優先順位の高い順に並べる
	modelsToTry := []string{
		"gemini-2.0-flash-exp",   // 本命
		"gemini-1.5-flash-002",   // 第2候補
		"gemini-1.5-pro-002",     // 第3候補
		"gemini-3-pro-preview",    // 予備
	}

	var lastErr error
	prompt := fmt.Sprintf("%s の商品説明を100文字程度で作成して", title)

	for _, modelName := range modelsToTry {
		// Cloud Runのログに現在試行中のモデルを出力
		log.Printf("DEBUG: モデル試行中... [%s]", modelName)

		model := client.GenerativeModel(modelName)
		resp, err := model.GenerateContent(ctx, genai.Text(prompt))

		if err == nil {
			// 成功した場合
			log.Printf("SUCCESS: 使用可能モデル発見! -> [%s]", modelName)
			if len(resp.Candidates) > 0 {
				return fmt.Sprintf("【使用モデル: %s】\n%v", modelName, resp.Candidates[0].Content.Parts[0]), nil
			}
			return "", fmt.Errorf("モデル %s は成功しましたが回答が空でした", modelName)
		}

		// 失敗した場合はエラーを記録して次へ
		log.Printf("INFO: モデル [%s] は使用不可: %v", modelName, err)
		lastErr = err
	}

	// すべてのモデルがダメだった場合
	return "", fmt.Errorf("【全滅】試した全てのモデルがNotFoundまたは権限エラーでした。最後のエラー: %w", lastErr)
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