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

// RiskAssessmentRequest represents the request payload for risk assessment.
type RiskAssessmentRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description" binding:"required"`
	Images      []string `json:"images"`
	ImageURLs   []string `json:"imageUrls"`
	Category    string   `json:"category"`
	Price       float64  `json:"price"`
	Condition   string   `json:"condition"`
}

// RiskAssessmentResponse represents the AI evaluation.
type RiskAssessmentResponse struct {
	RiskScore                float64  `json:"riskScore"`
	Reason                   string   `json:"reason"`
	Flags                    []string `json:"flags"`
	ImageMismatchScore       int      `json:"imageMismatchScore"`
	ImageMismatchReason      string   `json:"imageMismatchReason"`
	ReconstructedImageTitle  string   `json:"reconstructedImageTitle"`
	ClarityScore             int      `json:"clarityScore"`
	ClarityReason            string   `json:"clarityReason"`
	AuthenticityScore        int      `json:"authenticityScore"`
	AuthenticityReason       string   `json:"authenticityReason"`
	ReconstructedDescription string   `json:"reconstructedDescription"`
	CategoryFitScore         int      `json:"categoryFitScore"`
	CategoryReason           string   `json:"categoryReason"`
}

// RiskAssessment assesses the risk of a listing using AI with heuristics fallback.
func (h *HTTPHandler) RiskAssessment(c *gin.Context) {
	if h.vertexAIManager == nil || h.vertexAIManager.client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "VertexAI not initialized"})
		return
	}

	var req RiskAssessmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Detect extreme cases only (block before AI call).
	images := combineImages(req)
	if shouldBlockImmediate(req, images) {
		c.JSON(http.StatusOK, RiskAssessmentResponse{
			RiskScore:                0.95,
			Reason:                   "重大リスク検出のため診断を中止",
			Flags:                    []string{"高リスクキーワード検出 / 極度に短い説明"},
			ImageMismatchScore:       95,
			ImageMismatchReason:      "診断不可",
			ReconstructedImageTitle:  "",
			ClarityScore:             90,
			ClarityReason:            "説明が不十分",
			AuthenticityScore:        95,
			AuthenticityReason:       "信頼性が著しく低い",
			ReconstructedDescription: "",
			CategoryFitScore:         95,
			CategoryReason:           "診断中止",
		})
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

	prompt := buildRiskPrompt(req)

	// Low temperature for consistent evaluation
	temperature := float32(0.2)
	config := &genai.GenerateContentConfig{
		Temperature: &temperature,
	}

	resp, usedModel, lastErr := generateWithModelsAndConfig(ctx, h.vertexAIManager.client, attemptModels, genai.Text(prompt), config)
	if lastErr != nil {
		fmt.Printf("[ERROR] VertexAI risk assessment failed: %v\n", lastErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate risk assessment"})
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
		logger.Printf("\n=== RISK ASSESSMENT ===\n")
		logger.Printf("Prompt: %s\n", prompt)
		logger.Printf("Response text: %s\n", responseText)
		logger.Printf("Extracted JSON: %s\n", jsonText)
		logFile.Close()
	}

	var result RiskAssessmentResponse
	if err := json.Unmarshal([]byte(jsonText), &result); err != nil {
		fmt.Printf("[ERROR] Failed to parse risk assessment response: %v\n", err)
		fmt.Printf("[DEBUG] Response text: %s\n", responseText)
		fmt.Printf("[DEBUG] Extracted JSON: %s\n", jsonText)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse AI response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func buildRiskPrompt(req RiskAssessmentRequest) string {
	images := combineImages(req)
	imageList := "なし"
	if len(images) > 0 {
		imageList = strings.Join(images, ", ")
	}

	return fmt.Sprintf(`あなたは日本のフリマアプリの不正検知専門家です。以下の出品情報を厳格に解析し、指定のJSONのみ返してください。コードフェンス禁止、余計なテキスト禁止。

【出品情報】
タイトル: %s
説明: %s
カテゴリ: %s
価格: %.2f円
状態: %s
画像URL一覧: %s

【評価基準】全スコアは0-100(低い程リスク低=正しい)。厳格に採点すること。

1) 商品明瞭性 clarityScore:
   - 0-20: 非常に詳細で具体的(商品の特徴、寸法、状態を明記)
   - 21-40: 十分な情報あり(基本情報+α)
   - 41-60: 最低限の情報のみ
   - 61-80: 情報不足、曖昧
   - 81-100: 極めて不明瞭、ほぼ情報なし
   clarityReason に理由を20-40文字で簡潔に記載。

2) 真正性 authenticityScore:
   まずタイトル/説明から出品内容を20-60文字でAI再構成→reconstructedDescription
   - 0-20: 説明と完全一致、誠実な記述
   - 21-40: 概ね一致、小さな誇張あり
   - 41-60: やや不一致、誇張表現が目立つ
   - 61-80: 明確な矛盾あり、過大主張
   - 81-100: 完全に虚偽、詐欺の疑い強い
   authenticityReason に20-40文字で記載。

3) カテゴリ適合性 categoryFitScore:
   - 0-20: カテゴリと商品が完全一致
   - 21-40: 概ね適合、微妙なズレ
   - 41-60: やや不適合
   - 61-80: 明確な不一致
   - 81-100: 完全に不適合、意図的な誤分類の可能性
   categoryReason に20-40文字で記載。

4) 画像整合性 imageMismatchScore:
   画像URLから内容読取→AIで商品タイトル20-40文字生成→reconstructedImageTitle
   - 0-20: 画像とタイトル/説明が完全一致
   - 21-40: 概ね一致
   - 41-60: やや不一致(画像なし含む)
   - 61-80: 明確な不一致
   - 81-100: 全く異なる、詐欺の可能性
   画像URLなしの場合はスコア50。imageMismatchReason に20-40文字で記載。

5) 総合リスク riskScore (0.0-1.0):
   基本式: (clarityScore + authenticityScore + categoryFitScore + imageMismatchScore) / 250
   ※250で割ることで、平均スコア50なら0.8、平均30なら0.48となり適切な感度を確保
   
   追加調整(±0.2の範囲):
   - 価格が異常に低い/高い: +0.1~0.2
   - 説明の論理性が低い: +0.1
   - 緊急性を煽る表現: +0.1
   - 禁止ワード使用: +0.2
   - 特に問題なし: -0.05~0
   
   最終値は0.0-1.0に収める。reason に総評を30-60文字で記載。flags は懸念点を最大5件、各20文字以内で列挙。

出力JSON形式(厳守):
{
	"riskScore": number,
	"reason": "string",
	"flags": ["string"],
	"imageMismatchScore": int,
	"imageMismatchReason": "string",
	"reconstructedImageTitle": "string",
	"clarityScore": int,
	"clarityReason": "string",
	"authenticityScore": int,
	"authenticityReason": "string",
	"reconstructedDescription": "string",
	"categoryFitScore": int,
	"categoryReason": "string"
}

禁止事項: コードフェンス、追加キー、前置き/後置きの文章。`,
		req.Title, req.Description, req.Category, req.Price, req.Condition, imageList)
}

func shouldBlockImmediate(req RiskAssessmentRequest, images []string) bool {
	text := strings.ToLower(req.Title + " " + req.Description)
	descLen := len(strings.TrimSpace(req.Description))

	// Critical keywords only
	criticalKeywords := []string{"free gift card", "paypal only", "western union", "bitcoin", "counterfeit", "replica"}
	for _, kw := range criticalKeywords {
		if strings.Contains(text, kw) {
			return true
		}
	}

	// Extremely short (less than 20 chars) description
	if descLen < 20 {
		return true
	}

	return false
}

func heuristicRiskFlags(req RiskAssessmentRequest, images []string) []string {
	var flags []string
	text := strings.ToLower(req.Title + " " + req.Description)

	// Warning keywords (not blocking).
	warningKeywords := []string{"urgent", "limited time", "exclusive", "rare"}
	for _, kw := range warningKeywords {
		if strings.Contains(text, kw) {
			flags = append(flags, fmt.Sprintf("注意キーワード: %s", kw))
		}
	}

	// Suspiciously low price (warning only).
	if req.Price > 0 && req.Price < 500 && req.Condition != "中古" && req.Condition != "ジャンク" {
		flags = append(flags, "価格が低め")
	}

	return flags
}

func contains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func combineImages(req RiskAssessmentRequest) []string {
	if len(req.Images) > 0 {
		return req.Images
	}
	if len(req.ImageURLs) > 0 {
		return req.ImageURLs
	}
	return nil
}
