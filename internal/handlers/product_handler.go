package handlers

import (
	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/services"
	"github.com/gin-gonic/gin"
)

func GetProducts(c *gin.Context) {
	rows, _ := db.DB.Query("SELECT id, seller_id, title, price, image_url, is_sold FROM products ORDER BY created_at DESC")
	defer rows.Close()
	var products []models.Product
	for rows.Next() {
		var p models.Product
		rows.Scan(&p.ID, &p.SellerID, &p.Title, &p.Price, &p.ImageURL, &p.IsSold)
		products = append(products, p)
	}
	c.JSON(200, products)
}

func GetProductByID(c *gin.Context) {
	id := c.Param("id")
	var p models.Product
	db.DB.QueryRow("SELECT id, seller_id, title, description, price, image_url, is_sold FROM products WHERE id = ?", id).
		Scan(&p.ID, &p.SellerID, &p.Title, &p.Description, &p.Price, &p.ImageURL, &p.IsSold)
	c.JSON(200, p)
}

func CreateProduct(c *gin.Context) {
	var p models.Product
	c.ShouldBindJSON(&p)
	db.DB.Exec("INSERT INTO products (seller_id, title, description, price, image_url) VALUES (?, ?, ?, ?, ?)",
		p.SellerID, p.Title, p.Description, p.Price, p.ImageURL)
	c.JSON(201, p)
}

func PurchaseProduct(c *gin.Context) {
	id := c.Param("id")
	db.DB.Exec("UPDATE products SET is_sold = TRUE WHERE id = ?", id)
	c.JSON(200, gin.H{"status": "purchased"})
}

func GenerateAIDescription(c *gin.Context) {
	var req struct{ Title string `json:"title"` }
	c.ShouldBindJSON(&req)
	desc, _ := services.GenerateDescription(req.Title)
	c.JSON(200, gin.H{"description": desc})
}

func SuggestAIPrice(c *gin.Context) {
	var req struct{ Title string `json:"title"`; Description string `json:"description"` }
	c.ShouldBindJSON(&req)
	price, _ := services.SuggestPrice(req.Title, req.Description)
	c.JSON(200, gin.H{"suggestion": price})
}