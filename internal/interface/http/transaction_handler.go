package http

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Transaction model
type Transaction struct {
	ID              string `json:"id"`
	ItemID          string `json:"itemId"`
	BuyerID         string `json:"buyerId"`
	SellerID        string `json:"sellerId"`
	Price           int    `json:"price"`
	Quantity        int    `json:"quantity"`
	TransactionType string `json:"transactionType"`
	Warehouse       bool   `json:"warehouse"`
	Status          string `json:"status"`
	CreatedAt       string `json:"createdAt"`
}

// PriceHistory model
type PriceHistory struct {
	ID         string `json:"id"`
	ItemID     string `json:"itemId"`
	Price      int    `json:"price"`
	RecordedAt string `json:"recordedAt"`
}

// GetPriceHistory returns price history for an item (or product group)
func (h *HTTPHandler) GetPriceHistory(c *gin.Context) {
	itemID := c.Param("itemId")

	// 1) 商品のproduct_groupを取得
	var productGroup sql.NullString
	err := h.db.QueryRow(`SELECT product_group FROM items WHERE id = ?`, itemID).Scan(&productGroup)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var history []PriceHistory

	// 2) product_groupが設定されている場合、同じグループの全商品の取引履歴を統合
	if productGroup.Valid && productGroup.String != "" {
		txRows, err := h.db.Query(`
			SELECT t.id, t.item_id, t.price, t.created_at
			FROM transactions t
			JOIN items i ON t.item_id = i.id
			WHERE i.product_group = ? AND t.status = 'completed'
			ORDER BY t.created_at ASC
		`, productGroup.String)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer txRows.Close()

		for txRows.Next() {
			var ph PriceHistory
			if err := txRows.Scan(&ph.ID, &ph.ItemID, &ph.Price, &ph.RecordedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			history = append(history, ph)
		}

		// product_group商品の取引がない場合、シード履歴も統合
		if len(history) == 0 {
			rows, err := h.db.Query(`
				SELECT ph.id, ph.item_id, ph.price, ph.recorded_at 
				FROM price_history ph
				JOIN items i ON ph.item_id = i.id
				WHERE i.product_group = ?
				ORDER BY ph.recorded_at ASC
			`, productGroup.String)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			defer rows.Close()

			for rows.Next() {
				var ph PriceHistory
				if err := rows.Scan(&ph.ID, &ph.ItemID, &ph.Price, &ph.RecordedAt); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				history = append(history, ph)
			}
		}
	} else {
		// 3) product_groupなし: この商品単独の履歴を返す
		txRows, err := h.db.Query(`
			SELECT id, item_id, price, created_at
			FROM transactions
			WHERE item_id = ? AND status = 'completed'
			ORDER BY created_at ASC
		`, itemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer txRows.Close()

		for txRows.Next() {
			var ph PriceHistory
			if err := txRows.Scan(&ph.ID, &ph.ItemID, &ph.Price, &ph.RecordedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			history = append(history, ph)
		}

		// 取引が無い場合のみ、事前に記録されている price_history を返す（モック/シード用）
		if len(history) == 0 {
			rows, err := h.db.Query(`
				SELECT id, item_id, price, recorded_at 
				FROM price_history 
				WHERE item_id = ?
				ORDER BY recorded_at ASC
			`, itemID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			defer rows.Close()

			for rows.Next() {
				var ph PriceHistory
				if err := rows.Scan(&ph.ID, &ph.ItemID, &ph.Price, &ph.RecordedAt); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				history = append(history, ph)
			}
		}
	}

	c.JSON(http.StatusOK, history)
}

// CreateTransaction creates a new transaction record
func (h *HTTPHandler) CreateTransaction(c *gin.Context) {
	var req struct {
		ItemID          string `json:"itemId" binding:"required"`
		BuyerID         string `json:"buyerId" binding:"required"`
		SellerID        string `json:"sellerId" binding:"required"`
		Price           int    `json:"price" binding:"required"`
		Quantity        int    `json:"quantity"`
		TransactionType string `json:"transactionType" binding:"required"`
		Warehouse       bool   `json:"warehouse"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Quantity == 0 {
		req.Quantity = 1
	}

	txID := generateID()
	now := time.Now().Format("2006-01-02 15:04:05")

	_, err := h.db.Exec(`
		INSERT INTO transactions (id, item_id, buyer_id, seller_id, price, quantity, transaction_type, warehouse, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, txID, req.ItemID, req.BuyerID, req.SellerID, req.Price, req.Quantity, req.TransactionType, req.Warehouse, "completed", now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Also add to price history if it's an investment item
	var isInvestItem int
	err = h.db.QueryRow("SELECT is_invest_item FROM items WHERE id = ?", req.ItemID).Scan(&isInvestItem)
	if err == nil && isInvestItem == 1 {
		phID := generateID()
		_, _ = h.db.Exec(`
			INSERT INTO price_history (id, item_id, price, recorded_at)
			VALUES (?, ?, ?, ?)
		`, phID, req.ItemID, req.Price, now)
	}

	// Update item and user counters for purchase flows
	if req.TransactionType == "purchase" {
		_, _ = h.db.Exec("UPDATE items SET is_sold_out = 1 WHERE id = ?", req.ItemID)
		_, _ = h.db.Exec("UPDATE users SET listings_count = MAX(0, listings_count - 1) WHERE id = ?", req.SellerID)
		_, _ = h.db.Exec("UPDATE users SET transaction_count = transaction_count + 1 WHERE id IN (?, ?)", req.SellerID, req.BuyerID)
	}

	var tx Transaction
	err = h.db.QueryRow(`
		SELECT id, item_id, buyer_id, seller_id, price, quantity, transaction_type, warehouse, status, created_at
		FROM transactions WHERE id = ?
	`, txID).Scan(&tx.ID, &tx.ItemID, &tx.BuyerID, &tx.SellerID, &tx.Price, &tx.Quantity, &tx.TransactionType, &tx.Warehouse, &tx.Status, &tx.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tx)
}

// GetUserTransactions returns all transactions for a user (as buyer or seller)
func (h *HTTPHandler) GetUserTransactions(c *gin.Context) {
	userID := c.Param("userId")

	rows, err := h.db.Query(`
		SELECT t.id, t.item_id, t.buyer_id, t.seller_id, t.price, t.quantity, t.transaction_type, t.warehouse, t.status, t.created_at,
		       i.title, i.image_url
		FROM transactions t
		LEFT JOIN items i ON t.item_id = i.id
		WHERE t.buyer_id = ? OR t.seller_id = ?
		ORDER BY t.created_at DESC
	`, userID, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var transactions []gin.H
	for rows.Next() {
		var tx Transaction
		var itemTitle, itemImageURL sql.NullString

		if err := rows.Scan(&tx.ID, &tx.ItemID, &tx.BuyerID, &tx.SellerID, &tx.Price, &tx.Quantity,
			&tx.TransactionType, &tx.Warehouse, &tx.Status, &tx.CreatedAt, &itemTitle, &itemImageURL); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		transactions = append(transactions, gin.H{
			"id":              tx.ID,
			"itemId":          tx.ItemID,
			"buyerId":         tx.BuyerID,
			"sellerId":        tx.SellerID,
			"price":           tx.Price,
			"quantity":        tx.Quantity,
			"transactionType": tx.TransactionType,
			"warehouse":       tx.Warehouse,
			"status":          tx.Status,
			"createdAt":       tx.CreatedAt,
			"itemTitle":       itemTitle.String,
			"itemImageUrl":    itemImageURL.String,
		})
	}

	c.JSON(http.StatusOK, transactions)
}

// CompletePurchase completes a purchase and updates user statistics
func (h *HTTPHandler) CompletePurchase(c *gin.Context) {
	var req struct {
		ItemID    string `json:"itemId" binding:"required"`
		BuyerID   string `json:"buyerId" binding:"required"`
		SellerID  string `json:"sellerId" binding:"required"`
		Price     int    `json:"price" binding:"required"`
		Warehouse bool   `json:"warehouse"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// トランザクション開始
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	// 商品を売却済みに設定
	_, err = tx.Exec("UPDATE items SET is_sold_out = 1 WHERE id = ?", req.ItemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 売り手の出品数を-1
	_, err = tx.Exec("UPDATE users SET listings_count = MAX(0, listings_count - 1) WHERE id = ?", req.SellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 取引実績数を売り手・買い手の双方で+1
	_, err = tx.Exec("UPDATE users SET transaction_count = transaction_count + 1 WHERE id IN (?, ?)", req.SellerID, req.BuyerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// トランザクション記録を作成
	txID := generateID()
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = tx.Exec(`
		INSERT INTO transactions (id, item_id, buyer_id, seller_id, price, quantity, transaction_type, warehouse, status, created_at)
		VALUES (?, ?, ?, ?, ?, 1, 'purchase', ?, 'completed', ?)
	`, txID, req.ItemID, req.BuyerID, req.SellerID, req.Price, req.Warehouse, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// コミット
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 作成したトランザクション情報を取得してレスポンス
	var createdTx Transaction
	err = h.db.QueryRow(`
		SELECT id, item_id, buyer_id, seller_id, price, quantity, transaction_type, warehouse, status, created_at
		FROM transactions WHERE id = ?
	`, txID).Scan(&createdTx.ID, &createdTx.ItemID, &createdTx.BuyerID, &createdTx.SellerID, &createdTx.Price, &createdTx.Quantity, &createdTx.TransactionType, &createdTx.Warehouse, &createdTx.Status, &createdTx.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              createdTx.ID,
		"itemId":          createdTx.ItemID,
		"buyerId":         createdTx.BuyerID,
		"sellerId":        createdTx.SellerID,
		"price":           createdTx.Price,
		"quantity":        createdTx.Quantity,
		"transactionType": createdTx.TransactionType,
		"warehouse":       createdTx.Warehouse,
		"status":          createdTx.Status,
		"createdAt":       createdTx.CreatedAt,
		"message":         "Purchase completed successfully",
	})
} // GetAllTransactions returns all transactions (admin only)
func (h *HTTPHandler) GetAllTransactions(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT t.id, t.item_id, t.buyer_id, t.seller_id, t.price, t.quantity, t.transaction_type, t.warehouse, t.status, t.created_at,
		       i.title, i.image_url,
		       bu.name as buyer_name, su.name as seller_name
		FROM transactions t
		LEFT JOIN items i ON t.item_id = i.id
		LEFT JOIN users bu ON t.buyer_id = bu.id
		LEFT JOIN users su ON t.seller_id = su.id
		ORDER BY t.created_at DESC
	`)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var transactions []gin.H
	for rows.Next() {
		var tx Transaction
		var itemTitle, itemImageURL, buyerName, sellerName sql.NullString

		if err := rows.Scan(&tx.ID, &tx.ItemID, &tx.BuyerID, &tx.SellerID, &tx.Price, &tx.Quantity,
			&tx.TransactionType, &tx.Warehouse, &tx.Status, &tx.CreatedAt, &itemTitle, &itemImageURL, &buyerName, &sellerName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		transactions = append(transactions, gin.H{
			"id":              tx.ID,
			"itemId":          tx.ItemID,
			"buyerId":         tx.BuyerID,
			"sellerId":        tx.SellerID,
			"price":           tx.Price,
			"quantity":        tx.Quantity,
			"transactionType": tx.TransactionType,
			"warehouse":       tx.Warehouse,
			"status":          tx.Status,
			"createdAt":       tx.CreatedAt,
			"itemTitle":       itemTitle.String,
			"itemImageUrl":    itemImageURL.String,
			"buyerName":       buyerName.String,
			"sellerName":      sellerName.String,
		})
	}

	c.JSON(http.StatusOK, transactions)
}
