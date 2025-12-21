package http

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2/google"
)

// CreateListingRequest - 出品リクエスト
type CreateListingRequest struct {
	Title          string   `json:"title" binding:"required"`
	Description    string   `json:"description" binding:"required"`
	Price          int      `json:"price" binding:"required,min=300,max=9999999"`
	CategoryID     string   `json:"categoryId" binding:"required"`
	Condition      string   `json:"condition" binding:"required"`
	ImageUrls      []string `json:"imageUrls" binding:"required,min=1,max=10"`
	SellerID       string   `json:"sellerId" binding:"required"`
	IsInvestment   bool     `json:"isInvestment"`
	ShippingPaidBy string   `json:"shippingPaidBy"`
	ShippingMethod string   `json:"shippingMethod"`
	ShippingDays   string   `json:"shippingDays"`
}

// CreateInvestmentAssetRequest - 投資資産登録リクエスト
type CreateInvestmentAssetRequest struct {
	ItemID         string   `json:"itemId" binding:"required"`
	Grader         string   `json:"grader"`
	Grade          *float64 `json:"grade"`
	CertNumber     string   `json:"certNumber"`
	PurchaseDate   string   `json:"purchaseDate"`
	OriginalPrice  *int     `json:"originalPrice"`
	EstimatedValue *int     `json:"estimatedValue"`
}

// CreateWarehouseStorageRequest - 倉庫保管登録リクエスト
type CreateWarehouseStorageRequest struct {
	ItemID         string `json:"itemId" binding:"required"`
	WarehouseID    string `json:"warehouseId" binding:"required"`
	EstimatedValue int    `json:"estimatedValue" binding:"required"`
}

// CreateListing - 商品を出品
func (h *HTTPHandler) CreateListing(c *gin.Context) {
	var req CreateListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	itemID := generateID()
	isInvestItem := 0
	if req.IsInvestment {
		isInvestItem = 1
	}

	// 最初の画像を使用
	imageURL := ""
	if len(req.ImageUrls) > 0 {
		imageURL = req.ImageUrls[0]
	}

	// トランザクション開始
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	// 商品挿入
	_, err = tx.Exec(`
		INSERT INTO items (id, title, price, description, condition, category, image_url, is_sold_out, seller_id, is_invest_item)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?)
	`, itemID, req.Title, req.Price, req.Description, req.Condition, req.CategoryID, imageURL, req.SellerID, isInvestItem)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ユーザーの出品数をインクリメント
	_, err = tx.Exec(`
		UPDATE users SET listings_count = listings_count + 1 WHERE id = ?
	`, req.SellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update seller count: " + err.Error()})
		return
	}

	// トランザクション確定
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var item Item
	var description, condition, category sql.NullString
	err = tx.QueryRow(`
		SELECT id, title, price, description, condition, category, image_url, is_sold_out, seller_id, is_invest_item, view_count, like_count
		FROM items WHERE id = ?
	`, itemID).Scan(&item.ID, &item.Title, &item.Price, &description, &condition, &category, &item.ImageURL, &item.IsSoldOut, &item.SellerID, &item.IsInvestItem, &item.ViewCount, &item.LikeCount)
	if description.Valid {
		item.Description = &description.String
	}
	if condition.Valid {
		item.Condition = &condition.String
	}
	if category.Valid {
		item.Category = &category.String
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"itemId": itemID,
		"item":   item,
	})
}

// CreateInvestmentAsset - 投資資産情報を登録
func (h *HTTPHandler) CreateInvestmentAsset(c *gin.Context) {
	var req CreateInvestmentAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Grading Info作成
	gradingID := generateID()
	_, err := h.db.Exec(`
		INSERT INTO grading_info (id, item_id, grader, grade, cert_number)
		VALUES (?, ?, ?, ?, ?)
	`, gradingID, req.ItemID, req.Grader, req.Grade, req.CertNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Investment Asset作成
	assetID := generateID()
	_, err = h.db.Exec(`
		INSERT INTO investment_assets (id, item_id, purchase_date, original_price, estimated_value)
		VALUES (?, ?, ?, ?, ?)
	`, assetID, req.ItemID, req.PurchaseDate, req.OriginalPrice, req.EstimatedValue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"assetId":   assetID,
		"gradingId": gradingID,
	})
}

// CreateWarehouseStorage - 倉庫保管を登録
func (h *HTTPHandler) CreateWarehouseStorage(c *gin.Context) {
	var req CreateWarehouseStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	storageID := generateID()
	_, err := h.db.Exec(`
		INSERT INTO warehouse_storage (id, item_id, warehouse_id, estimated_value)
		VALUES (?, ?, ?, ?)
	`, storageID, req.ItemID, req.WarehouseID, req.EstimatedValue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"storageId": storageID,
	})
}

// GetInvestmentAsset - 投資資産情報を取得
func (h *HTTPHandler) GetInvestmentAsset(c *gin.Context) {
	itemID := c.Param("itemId")

	var grader string
	var grade, certNumber sql.NullString

	err := h.db.QueryRow(`
		SELECT grader, grade, cert_number FROM grading_info WHERE item_id = ?
	`, itemID).Scan(&grader, &grade, &certNumber)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Investment asset not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"grader":     grader,
		"grade":      grade.String,
		"certNumber": certNumber.String,
	})
}

// GetWarehouseStorage - 倉庫保管情報を取得
func (h *HTTPHandler) GetWarehouseStorage(c *gin.Context) {
	itemID := c.Param("itemId")

	var warehouseID string
	var estimatedValue int

	err := h.db.QueryRow(`
		SELECT warehouse_id, estimated_value FROM warehouse_storage WHERE item_id = ?
	`, itemID).Scan(&warehouseID, &estimatedValue)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Warehouse storage not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"warehouseId":    warehouseID,
		"estimatedValue": estimatedValue,
	})
}

// AnalyzeImageRequest - 画像をAIで解析
type AnalyzeImageRequest struct {
	ImageBase64 string `json:"imageBase64" binding:"required"`
	Prompt      string `json:"prompt"`
}

// AnalyzeImage - Geminiで画像を解析し、商品名/カテゴリ/状態コメントを返す
func (h *HTTPHandler) AnalyzeImage(c *gin.Context) {
	var req AnalyzeImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Load service account credentials
	credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsPath == "" {
		credsPath = "./secrets/service-account-key.json"
	}

	credsJSON, err := ioutil.ReadFile(credsPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load service account credentials", "details": err.Error()})
		return
	}

	// Get OAuth2 token
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, credsJSON, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse credentials", "details": err.Error()})
		return
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get OAuth2 token", "details": err.Error()})
		return
	}

	// Get project ID
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT_ID")
	if projectID == "" {
		projectID = "your-project-id"
	}

	cleanBase64 := req.ImageBase64
	if strings.Contains(cleanBase64, ",") {
		parts := strings.SplitN(req.ImageBase64, ",", 2)
		cleanBase64 = parts[1]
	}

	prompt := req.Prompt
	if prompt == "" {
		prompt = "You are a marketplace lister. Analyze the image and return JSON with keys: title (<=40 chars, Japanese), category (broad category name), conditionComment (short condition note). Return ONLY JSON."
	}

	// Vertex AI Gemini endpoint
	location := "us-central1"
	model := "gemini-1.5-flash-002"
	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		location, projectID, location, model)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": prompt},
					{"inline_data": map[string]interface{}{
						"mime_type": "image/png",
						"data":      cleanBase64,
					}},
				},
			},
		},
		"generation_config": map[string]interface{}{
			"temperature":     0.4,
			"maxOutputTokens": 2048,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	req2, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	client := &http.Client{}
	resp, err := client.Do(req2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := ioutil.ReadAll(resp.Body)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Vertex AI Gemini API error", "status": resp.StatusCode, "body": string(respBody)})
		return
	}

	var apiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	text := ""
	if len(apiResp.Candidates) > 0 && len(apiResp.Candidates[0].Content.Parts) > 0 {
		text = apiResp.Candidates[0].Content.Parts[0].Text
	}

	// Try to parse JSON from model output
	var parsed struct {
		Title            string `json:"title"`
		Category         string `json:"category"`
		ConditionComment string `json:"conditionComment"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err == nil && parsed.Title != "" {
		c.JSON(http.StatusOK, gin.H{
			"title":            parsed.Title,
			"category":         parsed.Category,
			"conditionComment": parsed.ConditionComment,
			"raw":              text,
		})
		return
	}

	// Fallback: return raw text
	c.JSON(http.StatusOK, gin.H{"raw": text})
}
