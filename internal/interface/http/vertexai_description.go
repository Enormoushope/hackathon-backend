package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"google.golang.org/genai"
)

// SuggestDescriptionRequest is the request for description suggestion.
type SuggestDescriptionRequest struct {
	Title       string `json:"title" binding:"required"`
	Condition   string `json:"condition" binding:"required"`
	Category    string `json:"category" binding:"required"`
	Description string `json:"description"`
}

// SuggestDescriptionResponse is the response for description suggestion.
type SuggestDescriptionResponse struct {
	Description string   `json:"description"`
	Highlights  []string `json:"highlights"`
}

// SuggestDescription suggests a description for a product.
func (h *HTTPHandler) SuggestDescription(c *gin.Context) {
	if h.vertexAIManager == nil || h.vertexAIManager.client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "VertexAI not initialized"})
		return
	}

	var req SuggestDescriptionRequest
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

	prompt := fmt.Sprintf(`あなたは日本のフリマアプリ出品文章作成専門家です。

以下の商品情報から、魅力的な商品説明文を作成してください。

【商品情報】
タイトル: %s
カテゴリ: %s
商品の状態: %s
現在の説明: %s

以下のJSON形式で回答してください:
{
  "description": "商品説明文（200～300字、購買意欲を高める内容）",
  "highlights": ["特徴1", "特徴2", "特徴3"]
}

厳格なルール:
- コードフェンスや前置き/後置きは入れない
- 純粋なJSONのみを返す
- キー名・型は指定通り`, req.Title, req.Category, req.Condition, req.Description)

	resp, usedModel, lastErr := generateWithModels(ctx, h.vertexAIManager.client, attemptModels, genai.Text(prompt))
	if lastErr != nil {
		// Return the underlying error detail to help surface VertexAI issues (e.g., auth/model not found)
		fmt.Printf("[ERROR] VertexAI description suggestion failed: %v (type: %T)\n", lastErr, lastErr)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate description suggestion",
			"details": lastErr.Error(),
		})
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
		logger.Printf("\n=== DESCRIPTION SUGGESTION ===\n")
		logger.Printf("Prompt: %s\n", prompt)
		logger.Printf("Response text: %s\n", responseText)
		logger.Printf("Extracted JSON: %s\n", jsonText)
		logFile.Close()
	}

	var result SuggestDescriptionResponse
	if err := json.Unmarshal([]byte(jsonText), &result); err != nil {
		fmt.Printf("[ERROR] Failed to parse description suggestion response: %v\n", err)
		fmt.Printf("[DEBUG] Response text: %s\n", responseText)
		fmt.Printf("[DEBUG] Extracted JSON: %s\n", jsonText)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse AI response"})
		return
	}

	c.JSON(http.StatusOK, result)
}
