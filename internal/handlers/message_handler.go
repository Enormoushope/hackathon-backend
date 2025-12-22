package handlers

import (
	"backend/internal/db"
	"backend/internal/models"
	"fmt"
	"net/http"
	"strconv"

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
	productIDStr := c.Query("product_id")
	user1 := c.Query("user1")
	user2 := c.Query("user2")

	// 型情報をログ出力
	logMsg := "[GetChatHistory] product_id: " + productIDStr + " (" + fmt.Sprintf("%T", productIDStr) + ") " +
		"user1: " + user1 + " (" + fmt.Sprintf("%T", user1) + ") " +
		"user2: " + user2 + " (" + fmt.Sprintf("%T", user2) + ")"
	println(logMsg)

	// product_idをintに変換
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_idは数値で指定してください", "product_id": productIDStr, "type": fmt.Sprintf("%T", productIDStr)})
		return
	}

	rows, err := db.DB.Query(`
		   SELECT id, product_id, sender_id, receiver_id, content, created_at 
		   FROM messages 
		   WHERE product_id = ? 
		   AND ((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?))
		   ORDER BY created_at ASC`,
		productID, user1, user2, user2, user1,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DBからのSELECTに失敗: " + err.Error(), "product_id": productID, "product_id_type": fmt.Sprintf("%T", productID), "user1": user1, "user2": user2})
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
