package handlers

import (
	"backend/internal/db"
	"backend/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ToggleLike: いいねの登録と解除を切り替える
func ToggleLike(c *gin.Context) {
	var l models.Like
	if err := c.ShouldBindJSON(&l); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無効なリクエストです"})
		return
	}

	// すでにいいねしているか確認
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = ? AND product_id = ?)"
	err := db.DB.QueryRow(query, l.UserID, l.ProductID).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "データベースエラー"})
		return
	}

	if exists {
		// すでにある場合は削除（解除）
		_, err = db.DB.Exec("DELETE FROM likes WHERE user_id = ? AND product_id = ?", l.UserID, l.ProductID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解除に失敗しました"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "unliked", "is_liked": false})
	} else {
		// ない場合は挿入（登録）
		_, err = db.DB.Exec("INSERT INTO likes (user_id, product_id) VALUES (?, ?)", l.UserID, l.ProductID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "登録に失敗しました"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "liked", "is_liked": true})
	}
}

// CheckLikeStatus: フロントエンド表示時に「いいね済」かどうかを判定する
func CheckLikeStatus(c *gin.Context) {
	userID := c.Query("user_id")
	productID := c.Query("product_id")

	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = ? AND product_id = ?)"
	db.DB.QueryRow(query, userID, productID).Scan(&exists)

	c.JSON(http.StatusOK, gin.H{"is_liked": exists})
}