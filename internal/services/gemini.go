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
    
    // 1. クライアント作成のチェック
    client, err := GetGeminiClient(ctx)
    if err != nil {
        return "", fmt.Errorf("【診断:1 クライアント作成失敗】: %w", err)
    }
    defer client.Close()

    // 2. モデル取得のチェック
    // ※今 Model Garden で見えている一番新しい名前をここに入れてください
    modelName := "gemini-3-pro-preview" 
    model := client.GenerativeModel(modelName)
    if model == nil {
        return "", fmt.Errorf("【診断:2 モデル指定エラー】: モデル名 '%s' が取得できませんでした", modelName)
    }

    // 3. 実際にAIに送ってみる
    prompt := fmt.Sprintf("%s の商品説明を100文字程度で作成して", title)
    resp, err := model.GenerateContent(ctx, genai.Text(prompt))
    
    if err != nil {
        // ここが一番重要：404なのか403（権限）なのかを判別
        return "", fmt.Errorf("【診断:3 生成APIエラー】モデル(%s)で失敗: %w", modelName, err)
    }

    // 4. 結果の解析
    if len(resp.Candidates) == 0 {
        return "", fmt.Errorf("【診断:4 応答なし】AIからの回答が空でした")
    }

    return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
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