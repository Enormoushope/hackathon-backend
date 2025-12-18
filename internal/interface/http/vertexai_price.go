package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/genai"
)

// SuggestPriceRequest is the request for price suggestion.
type SuggestPriceRequest struct {
	Title       string `json:"title" binding:"required"`
	Condition   string `json:"condition" binding:"required"`
	Category    string `json:"category" binding:"required"`
	Description string `json:"description"`
}

// SuggestPriceResponse is the response for price suggestion.
type SuggestPriceResponse struct {
	SuggestedPrice int    `json:"suggestedPrice"`
	Reasoning      string `json:"reasoning"`
	PriceRange     struct {
		Min int `json:"min"`
		Max int `json:"max"`
	} `json:"priceRange"`
}

// SuggestPrice suggests a price for a product.
func (h *HTTPHandler) SuggestPrice(c *gin.Context) {
	if h.vertexAIManager == nil || h.vertexAIManager.client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "VertexAI not initialized"})
		return
	}

	var req SuggestPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	attemptModels := []string{
		"gemini-1.5-pro-001",
		"gemini-1.5-pro-002",
		"gemini-2.0-flash-001",
		"gemini-2.0-pro-exp-02-05",
		"gemini-1.5-flash-001",
	}
	usedModel := ""

	prompt := fmt.Sprintf(`あなたは日本のフリマアプリの価格査定専門家です。

以下の商品情報から、適正な出品価格を提案してください。

【商品情報】
タイトル: %s
カテゴリ: %s
商品の状態: %s
説明: %s

以下のJSON形式で回答してください:
{
  "suggestedPrice": 数値（推奨価格・日本円),
  "reasoning": "価格を決めた理由（100字以内）",
  "priceRange": {
    "min": 最安値,
    "max": 最高値
  }
}

注意: 次の厳格なフォーマット規則に従ってください。
- コードフェンス(三連バッククォートやjsonコードフェンス)は使わない
- 前後の説明文やラベルは一切入れない
- 出力は { で始まり } で終わる純粋なJSONのみ
- キー名は指定したとおり、型も厳守`, req.Title, req.Category, req.Condition, req.Description)

	resp, usedModel, lastErr := generateWithModels(ctx, h.vertexAIManager.client, attemptModels, genai.Text(prompt))
	if lastErr != nil {
		fmt.Printf("[ERROR] VertexAI price suggestion failed: %v\n", lastErr)
		fmt.Printf("[ERROR] Error type: %T\n", lastErr)
		fmt.Printf("[ERROR] Error string: %s\n", lastErr.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate price suggestion", "details": lastErr.Error()})
		return
	}

	if usedModel != "" && usedModel != attemptModels[0] {
		log.Printf("[INFO] VertexAI fallback model used: %s", usedModel)
	}

	responseText := collectPartsText(resp)
	jsonText := extractJSON(responseText)

	logFile, _ := os.OpenFile("C:\\Users\\xyz77\\hackathon\\backend\\ai_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if logFile != nil {
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("\n=== PRICE SUGGESTION ===\n")
		logger.Printf("Prompt: %s\n", prompt)
		logger.Printf("Response text: %s\n", responseText)
		logger.Printf("Extracted JSON: %s\n", jsonText)
		logFile.Close()
	}

	var result SuggestPriceResponse
	if err := json.Unmarshal([]byte(jsonText), &result); err != nil {
		fmt.Printf("[ERROR] Failed to parse price suggestion response: %v\n", err)
		fmt.Printf("[DEBUG] Response text: %s\n", responseText)
		fmt.Printf("[DEBUG] Extracted JSON: %s\n", jsonText)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse AI response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// collectPartsText concatenates text parts from a response.
func collectPartsText(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return ""
	}
	var b strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		b.WriteString(part.Text)
	}
	return b.String()
}

// generateWithModels runs the models in order and returns the first successful response.
func generateWithModels(ctx context.Context, client *genai.Client, models []string, input []*genai.Content) (*genai.GenerateContentResponse, string, error) {
	return generateWithModelsAndConfig(ctx, client, models, input, nil)
}

func generateWithModelsAndConfig(ctx context.Context, client *genai.Client, models []string, input []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, string, error) {
	var lastErr error
	for _, modelName := range models {
		fmt.Printf("[DEBUG] Attempting model: %s\n", modelName)
		resp, err := client.Models.GenerateContent(ctx, modelName, input, config)
		if err != nil {
			fmt.Printf("[DEBUG] Model %s failed: %v (type: %T)\n", modelName, err, err)
			lastErr = err
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				fmt.Printf("[DEBUG] Model %s not found, trying next...\n", modelName)
				continue
			}
			return nil, "", err
		}
		fmt.Printf("[DEBUG] Model %s succeeded\n", modelName)
		return resp, modelName, nil
	}
	if lastErr != nil {
		fmt.Printf("[ERROR] All models failed. Last error: %v\n", lastErr)
		return nil, "", lastErr
	}
	return nil, "", fmt.Errorf("no model response")
}
