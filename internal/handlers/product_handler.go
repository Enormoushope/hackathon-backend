package handlers

import (
	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/services" // servicesをインポート
	"github.com/gin-gonic/gin"
	"net/http"
)

// --- 商品一覧取得 ---
func GetProducts(c *gin.Context) {
	rows, err := db.DB.Query("SELECT id, seller_id, title, price, image_url, is_sold FROM products ORDER BY created_at DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "商品一覧の取得に失敗しました"})
		return
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.SellerID, &p.Title, &p.Price, &p.ImageURL, &p.IsSold); err != nil {
			continue
		}
		products = append(products, p)
	}
	c.JSON(http.StatusOK, products)
}

// --- 商品詳細取得 ---
func GetProductByID(c *gin.Context) {
	id := c.Param("id")
	var p models.Product
	err := db.DB.QueryRow("SELECT id, seller_id, title, description, price, image_url, is_sold FROM products WHERE id = ?", id).
		Scan(&p.ID, &p.SellerID, &p.Title, &p.Description, &p.Price, &p.ImageURL, &p.IsSold)
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商品が見つかりませんでした"})
		return
	}
	c.JSON(http.StatusOK, p)
}

// --- 新規出品 ---
func CreateProduct(c *gin.Context) {
	var p models.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "リクエスト形式が正しくありません"})
		return
	}

	_, err := db.DB.Exec("INSERT INTO products (seller_id, title, description, price, image_url) VALUES (?, ?, ?, ?, ?)",
		p.SellerID, p.Title, p.Description, p.Price, p.ImageURL)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "データベースへの保存に失敗しました: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

// --- 商品購入（売り切れ更新） ---
func PurchaseProduct(c *gin.Context) {
	id := c.Param("id")
	_, err := db.DB.Exec("UPDATE products SET is_sold = TRUE WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "購入処理に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "purchased"})
}

// --- AI商品説明生成 ---
// backend/internal/handlers/product_handler.go

func GenerateAIDescription(c *gin.Context) {
    var req struct{ Title string `json:"title"` }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "JSON形式が不正です"})
        return
    }

    desc, err := services.GenerateDescription(req.Title)
    if err != nil {
        // 🔴 err.Error() をそのまま返すように変更
        c.JSON(500, gin.H{"error": "内部エラー: " + err.Error()})
        return
    }

    c.JSON(200, gin.H{"description": desc})
}

// --- AI価格査定 ---
func SuggestAIPrice(c *gin.Context) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "入力内容が不足しています"})
		return
	}

	// services/gemini.go に SuggestPrice 関数がある前提
	price, err := services.SuggestPrice(req.Title, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "価格査定に失敗しました: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"suggestion": price})
}