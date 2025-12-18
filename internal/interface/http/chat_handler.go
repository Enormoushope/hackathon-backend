package http

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateConversation creates a new conversation
func (h *HTTPHandler) CreateConversation(c *gin.Context) {
	var req struct {
		ItemID   string `json:"itemId" binding:"required"`
		BuyerID  string `json:"buyerId" binding:"required"`
		SellerID string `json:"sellerId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if conversation already exists
	var existingID string
	err := h.db.QueryRow("SELECT id FROM conversations WHERE item_id = ? AND buyer_id = ? AND seller_id = ?",
		req.ItemID, req.BuyerID, req.SellerID).Scan(&existingID)
	if err == nil {
		// Return existing conversation
		var conv Conversation
		h.db.QueryRow("SELECT id, item_id, buyer_id, seller_id, created_at, updated_at FROM conversations WHERE id = ?", existingID).
			Scan(&conv.ID, &conv.ItemID, &conv.BuyerID, &conv.SellerID, &conv.CreatedAt, &conv.UpdatedAt)
		c.JSON(http.StatusOK, conv)
		return
	}

	// Create new conversation
	convID := generateID()
	_, err = h.db.Exec("INSERT INTO conversations (id, item_id, buyer_id, seller_id) VALUES (?, ?, ?, ?)",
		convID, req.ItemID, req.BuyerID, req.SellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var conv Conversation
	h.db.QueryRow("SELECT id, item_id, buyer_id, seller_id, created_at, updated_at FROM conversations WHERE id = ?", convID).
		Scan(&conv.ID, &conv.ItemID, &conv.BuyerID, &conv.SellerID, &conv.CreatedAt, &conv.UpdatedAt)

	c.JSON(http.StatusCreated, conv)
}

// GetConversation returns a conversation by ID
func (h *HTTPHandler) GetConversation(c *gin.Context) {
	id := c.Param("id")
	var conv Conversation
	err := h.db.QueryRow("SELECT id, item_id, buyer_id, seller_id, created_at, updated_at FROM conversations WHERE id = ?", id).
		Scan(&conv.ID, &conv.ItemID, &conv.BuyerID, &conv.SellerID, &conv.CreatedAt, &conv.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, conv)
}

// GetConversationsByUser returns conversations for a user
func (h *HTTPHandler) GetConversationsByUser(c *gin.Context) {
	userID := c.Query("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
		return
	}

	rows, err := h.db.Query("SELECT id, item_id, buyer_id, seller_id, created_at, updated_at FROM conversations WHERE buyer_id = ? OR seller_id = ? ORDER BY updated_at DESC",
		userID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var convList []Conversation
	for rows.Next() {
		var conv Conversation
		if err := rows.Scan(&conv.ID, &conv.ItemID, &conv.BuyerID, &conv.SellerID, &conv.CreatedAt, &conv.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		convList = append(convList, conv)
	}

	c.JSON(http.StatusOK, convList)
}

// SendMessage sends a message in a conversation
func (h *HTTPHandler) SendMessage(c *gin.Context) {
	var req struct {
		ConversationID string `json:"conversationId" binding:"required"`
		SenderID       string `json:"senderId" binding:"required"`
		Content        string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msgID := generateID()
	_, err := h.db.Exec("INSERT INTO messages (id, conversation_id, sender_id, content) VALUES (?, ?, ?, ?)",
		msgID, req.ConversationID, req.SenderID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update conversation updated_at
	h.db.Exec("UPDATE conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", req.ConversationID)

	var msg Message
	h.db.QueryRow("SELECT id, conversation_id, sender_id, content, created_at FROM messages WHERE id = ?", msgID).
		Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content, &msg.CreatedAt)

	c.JSON(http.StatusCreated, msg)
}

// GetMessages returns messages in a conversation
func (h *HTTPHandler) GetMessages(c *gin.Context) {
	convID := c.Param("id")

	rows, err := h.db.Query("SELECT id, conversation_id, sender_id, content, created_at FROM messages WHERE conversation_id = ? ORDER BY created_at ASC", convID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var msgList []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content, &msg.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		msgList = append(msgList, msg)
	}

	c.JSON(http.StatusOK, msgList)
}
