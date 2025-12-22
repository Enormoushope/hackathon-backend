package handlers

import (
	"backend/internal/db"
	"backend/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// メッセージ送信
func SendMessage(c *gin.Context) {
	var m models.Message
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストのJSONパースに失敗: " + err.Error()})
		return
	}

	result, err := db.DB.Exec(
		"INSERT INTO messages (product_id, sender_id, receiver_id, content) VALUES (?, ?, ?, ?)",
		m.ProductID, m.SenderID, m.ReceiverID, m.Content,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DBへのINSERTに失敗: " + err.Error()})
		return
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LastInsertId取得に失敗: " + err.Error()})
		return
	}
	m.ID = int(lastID)
	c.JSON(http.StatusCreated, m)
}

// 特定の商品・ユーザー間のチャット履歴取得
func GetChatHistory(c *gin.Context) {
	productID := c.Query("product_id")
	user1 := c.Query("user1") // ログインユーザー
	user2 := c.Query("user2") // 相手（出品者または購入希望者）

	rows, err := db.DB.Query(`
	       SELECT id, product_id, sender_id, receiver_id, content, created_at 
	       FROM messages 
	       WHERE product_id = ? 
	       AND ((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?))
	       ORDER BY created_at ASC`,
		productID, user1, user2, user2, user1,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DBからのSELECTに失敗: " + err.Error()})
		return
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.ProductID, &m.SenderID, &m.ReceiverID, &m.Content, &m.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "行のScanに失敗: " + err.Error()})
			return
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rows.Err(): " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, messages)
}
