package http

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// （カテゴリ関連は以前の挙動に戻すため、ヘルパーは削除）

// GetItems returns all items with optional search, filters and priority ranking
func (h *HTTPHandler) GetItems(c *gin.Context) {
	searchQuery := c.Query("query")
	categoryCode := c.Query("categoryCode") // カテゴリコードでフィルター
	minPrice := c.Query("minPrice")
	maxPrice := c.Query("maxPrice")
	investOnly := c.Query("investOnly") // "true" for investment items only
	normalOnly := c.Query("normalOnly") // "true" for normal items only
	sortBy := c.Query("sortBy")         // "price_asc", "price_desc", "newest"

	query := `
		SELECT 
			i.id, i.title, i.price, i.description, i.condition, i.category, i.image_url, i.is_sold_out, i.seller_id, i.is_invest_item,
			i.view_count, i.like_count, (SELECT COUNT(*) FROM item_reactions r WHERE r.item_id = i.id AND r.reaction_type = 'watch') AS watch_count, i.product_group,
			u.rating, u.follower_count, u.listings_count,
			-- Priority Score Algorithm
			(
				COALESCE(u.rating, 3.0) * 20 +
				COALESCE(u.follower_count, 0) * 2 +
				COALESCE(u.listings_count, 0) * 1.5 +
				COALESCE(i.like_count, 0) * 10 +
				COALESCE(i.view_count, 0) * 0.5
			) as priority_score
		FROM items i
		LEFT JOIN users u ON i.seller_id = u.id
		WHERE i.is_sold_out = 0
	`

	var params []interface{}

	// Enhanced search filter - supports partial matching and multiple keywords
	if searchQuery != "" {
		// Split search query into individual keywords (space-separated)
		keywords := strings.Fields(searchQuery)

		if len(keywords) > 0 {
			searchConditions := []string{}

			for _, keyword := range keywords {
				// Each keyword is matched as a partial match (case-insensitive with LIKE)
				searchConditions = append(searchConditions, "(LOWER(i.title) LIKE ? OR LOWER(i.id) LIKE ?)")
				searchParam := "%" + strings.ToLower(keyword) + "%"
				params = append(params, searchParam, searchParam)
			}

			// Combine all keyword conditions with OR (at least one keyword must match)
			query += " AND (" + strings.Join(searchConditions, " OR ") + ")"
		}
	}

	// Category filter by code (完全一致のみ)
	if categoryCode != "" {
		query += ` AND i.category = ?`
		params = append(params, categoryCode)
	}

	// Price range filter
	if minPrice != "" {
		query += ` AND i.price >= ?`
		params = append(params, minPrice)
	}
	if maxPrice != "" {
		query += ` AND i.price <= ?`
		params = append(params, maxPrice)
	}

	// Investment/Normal filter
	if investOnly == "true" {
		query += ` AND i.is_invest_item = 1`
	} else if normalOnly == "true" {
		query += ` AND i.is_invest_item = 0`
	}

	// Sorting
	switch sortBy {
	case "price_asc":
		query += ` ORDER BY i.price ASC, i.id`
	case "price_desc":
		query += ` ORDER BY i.price DESC, i.id`
	case "newest":
		query += ` ORDER BY i.id DESC`
	default:
		query += ` ORDER BY priority_score DESC, i.id`
	}

	rows, err := h.db.Query(query, params...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var itemList []Item
	for rows.Next() {
		var item Item
		var soldOut, investItem int
		var viewCount, likeCount, watchCount sql.NullInt64
		var sellerRating sql.NullFloat64
		var sellerFollowers, sellerListings sql.NullInt64
		var priorityScore float64
		var productGroup, description, condition, category sql.NullString

		if err := rows.Scan(
			&item.ID, &item.Title, &item.Price, &description, &condition, &category, &item.ImageURL, &soldOut, &item.SellerID, &investItem,
			&viewCount, &likeCount, &watchCount, &productGroup,
			&sellerRating, &sellerFollowers, &sellerListings,
			&priorityScore,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		item.IsSoldOut = soldOut == 1
		isInvest := investItem == 1
		item.IsInvestItem = &isInvest
		if description.Valid {
			item.Description = &description.String
		}
		if condition.Valid {
			item.Condition = &condition.String
		}
		if category.Valid {
			item.Category = &category.String
		}
		if viewCount.Valid {
			vc := int(viewCount.Int64)
			item.ViewCount = &vc
		}
		if likeCount.Valid {
			lc := int(likeCount.Int64)
			item.LikeCount = &lc
		}
		if watchCount.Valid {
			wc := int(watchCount.Int64)
			item.WatchCount = &wc
		}
		if sellerRating.Valid {
			item.SellerRating = &sellerRating.Float64
		}
		if productGroup.Valid && productGroup.String != "" {
			item.ProductGroup = &productGroup.String
		}
		itemList = append(itemList, item)
	}

	c.JSON(http.StatusOK, itemList)
}

// GetItemByID returns an item by ID with seller rating
func (h *HTTPHandler) GetItemByID(c *gin.Context) {
	id := c.Param("id")

	var item Item
	var soldOut, investItem int
	var viewCount, likeCount, watchCount sql.NullInt64
	var sellerRating sql.NullFloat64
	var productGroup, description, condition, category sql.NullString
	err := h.db.QueryRow(`
		SELECT i.id, i.title, i.price, i.description, i.condition, i.category, i.image_url, i.is_sold_out, i.seller_id, i.is_invest_item, 
		       i.view_count, i.like_count, (SELECT COUNT(*) FROM item_reactions r WHERE r.item_id = i.id AND r.reaction_type = 'watch') AS watch_count, i.product_group, u.rating
		FROM items i
		LEFT JOIN users u ON i.seller_id = u.id
		WHERE i.id = ?
	`, id).
		Scan(&item.ID, &item.Title, &item.Price, &description, &condition, &category, &item.ImageURL, &soldOut, &item.SellerID, &investItem, &viewCount, &likeCount, &watchCount, &productGroup, &sellerRating)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	item.IsSoldOut = soldOut == 1
	item.IsSoldOut = soldOut == 1
	isInvest := investItem == 1
	item.IsInvestItem = &isInvest
	if description.Valid {
		item.Description = &description.String
	}
	if condition.Valid {
		item.Condition = &condition.String
	}
	if category.Valid {
		item.Category = &category.String
	}
	if viewCount.Valid {
		vc := int(viewCount.Int64)
		item.ViewCount = &vc
	}
	if likeCount.Valid {
		lc := int(likeCount.Int64)
		item.LikeCount = &lc
	}
	if sellerRating.Valid {
		item.SellerRating = &sellerRating.Float64
	}
	if productGroup.Valid && productGroup.String != "" {
		item.ProductGroup = &productGroup.String
	}

	c.JSON(http.StatusOK, item)
}

// IncrementViewCount increments the view count for an item
func (h *HTTPHandler) IncrementViewCount(c *gin.Context) {
	id := c.Param("id")

	result, err := h.db.Exec("UPDATE items SET view_count = view_count + 1 WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	// Return updated item with new view count
	var item Item
	var soldOut, investItem int
	var viewCount, likeCount, watchCount sql.NullInt64
	var sellerRating sql.NullFloat64
	err = h.db.QueryRow(`
		SELECT i.id, i.title, i.price, i.image_url, i.is_sold_out, i.seller_id, i.is_invest_item, 
		       i.view_count, i.like_count, (SELECT COUNT(*) FROM item_reactions r WHERE r.item_id = i.id AND r.reaction_type = 'watch') AS watch_count, u.rating
		FROM items i
		LEFT JOIN users u ON i.seller_id = u.id
		WHERE i.id = ?
	`, id).
		Scan(&item.ID, &item.Title, &item.Price, &item.ImageURL, &soldOut, &item.SellerID, &investItem, &viewCount, &likeCount, &watchCount, &sellerRating)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	item.IsSoldOut = soldOut == 1
	isInvest := investItem == 1
	item.IsInvestItem = &isInvest
	if viewCount.Valid {
		vc := int(viewCount.Int64)
		item.ViewCount = &vc
	}
	if likeCount.Valid {
		lc := int(likeCount.Int64)
		item.LikeCount = &lc
	}
	if watchCount.Valid {
		wc := int(watchCount.Int64)
		item.WatchCount = &wc
	}
	if sellerRating.Valid {
		item.SellerRating = &sellerRating.Float64
	}

	c.JSON(http.StatusOK, item)
}

// AddReaction adds a reaction (like/watch) to an item
func (h *HTTPHandler) AddReaction(c *gin.Context) {
	var req struct {
		ItemID       string `json:"itemId" binding:"required"`
		UserID       string `json:"userId" binding:"required"`
		ReactionType string `json:"reactionType" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ReactionType != "like" && req.ReactionType != "watch" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reactionType must be 'like' or 'watch'"})
		return
	}

	reactionID := generateID()
	_, err := h.db.Exec("INSERT OR REPLACE INTO item_reactions (id, item_id, user_id, reaction_type) VALUES (?, ?, ?, ?)",
		reactionID, req.ItemID, req.UserID, req.ReactionType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update item like count
	if req.ReactionType == "like" {
		h.db.Exec("UPDATE items SET like_count = (SELECT COUNT(*) FROM item_reactions WHERE item_id = ? AND reaction_type = 'like') WHERE id = ?",
			req.ItemID, req.ItemID)
	}

	var reaction ItemReaction
	h.db.QueryRow("SELECT id, item_id, user_id, reaction_type, created_at FROM item_reactions WHERE item_id = ? AND user_id = ? AND reaction_type = ?",
		req.ItemID, req.UserID, req.ReactionType).
		Scan(&reaction.ID, &reaction.ItemID, &reaction.UserID, &reaction.ReactionType, &reaction.CreatedAt)

	c.JSON(http.StatusCreated, reaction)
}

// RemoveReaction removes a reaction from an item
func (h *HTTPHandler) RemoveReaction(c *gin.Context) {
	var req struct {
		ItemID       string `json:"itemId" binding:"required"`
		UserID       string `json:"userId" binding:"required"`
		ReactionType string `json:"reactionType" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.db.Exec("DELETE FROM item_reactions WHERE item_id = ? AND user_id = ? AND reaction_type = ?",
		req.ItemID, req.UserID, req.ReactionType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reaction not found"})
		return
	}

	// Update item like count
	if req.ReactionType == "like" {
		h.db.Exec("UPDATE items SET like_count = (SELECT COUNT(*) FROM item_reactions WHERE item_id = ? AND reaction_type = 'like') WHERE id = ?",
			req.ItemID, req.ItemID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reaction removed"})
}

// GetItemReactions returns all reactions for an item
func (h *HTTPHandler) GetItemReactions(c *gin.Context) {
	itemID := c.Param("itemId")

	rows, err := h.db.Query("SELECT id, item_id, user_id, reaction_type, created_at FROM item_reactions WHERE item_id = ?", itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var reactions []ItemReaction
	for rows.Next() {
		var r ItemReaction
		if err := rows.Scan(&r.ID, &r.ItemID, &r.UserID, &r.ReactionType, &r.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		reactions = append(reactions, r)
	}

	c.JSON(http.StatusOK, reactions)
}

// GetUserReactions returns all reactions by a user
func (h *HTTPHandler) GetUserReactions(c *gin.Context) {
	userID := c.Param("userId")
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	rows, err := h.db.Query("SELECT id, item_id, user_id, reaction_type, created_at FROM item_reactions WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?", userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var reactions []ItemReaction
	for rows.Next() {
		var r ItemReaction
		if err := rows.Scan(&r.ID, &r.ItemID, &r.UserID, &r.ReactionType, &r.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		reactions = append(reactions, r)
	}

	c.JSON(http.StatusOK, gin.H{"reactions": reactions})
}
