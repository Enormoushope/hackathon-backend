// User struct (NULL許容カラムはポインタ型)
type User struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	AvatarURL        *string  `json:"avatarUrl"`
	Bio              *string  `json:"bio"`
	Rating           *float64 `json:"rating"`
	SellingCount     *int     `json:"sellingCount"`
	FollowerCount    *int     `json:"followerCount"`
	ReviewCount      *int     `json:"reviewCount"`
	TransactionCount *int     `json:"transactionCount"`
}
package http

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetCurrentUser returns the authenticated user's profile by Firebase UID
func (h *HTTPHandler) GetCurrentUser(c *gin.Context) {
	uidValue, exists := c.Get("uid")
	if !exists || uidValue == "" {
		// Firebase未初期化の場合、テスト用に固定UIDを使用
		fmt.Println("[WARN] No uid in context - using test user")
		uidValue = "18oYncIdc3UuvZneYQQ4j2II23A2"
	}

	uid := uidValue.(string)
	fmt.Printf("[DEBUG] GetCurrentUser: uid=%s\n", uid)
	var u User
	var isAdmin int
		err := h.db.QueryRow(`
			SELECT id, username, avatar_url, bio, rating, listings_count, follower_count, review_count, transaction_count, is_admin
			FROM users WHERE id = ?`, uid).
			Scan(&u.ID, &u.Name, &u.AvatarURL, &u.Bio, &u.Rating, &u.SellingCount, &u.FollowerCount, &u.ReviewCount, &u.TransactionCount, &isAdmin)
	if err == sql.ErrNoRows {
		fmt.Printf("[DEBUG] User not found in DB: uid=%s\n", uid)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		fmt.Printf("[ERROR] Query failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 動的に出品数を再計算（データベースとの同期を確認）
	var actualSellingCount int
	err = h.db.QueryRow("SELECT COUNT(*) FROM items WHERE seller_id = ? AND is_sold_out = 0", uid).Scan(&actualSellingCount)
	if err == nil {
	    // listings_countカラムに合わせて更新
	    if u.SellingCount != nil && actualSellingCount != *u.SellingCount {
		fmt.Printf("[SYNC] Updating listings_count for user %s: %d -> %d\n", uid, *u.SellingCount, actualSellingCount)
		_, updateErr := h.db.Exec("UPDATE users SET listings_count = ? WHERE id = ?", actualSellingCount, uid)
		if updateErr != nil {
		    fmt.Printf("[ERROR] Failed to sync listings_count: %v\n", updateErr)
		} else {
		    u.SellingCount = &actualSellingCount
		}
	    }
	}

	// Return user with isAdmin field
	response := gin.H{
		"id":               u.ID,
		"name":             u.Name,
		"avatarUrl":        u.AvatarURL,
		"bio":              u.Bio,
		"rating":           u.Rating,
		"sellingCount":     u.SellingCount,
		"followerCount":    u.FollowerCount,
		"reviewCount":      u.ReviewCount,
		"transactionCount": u.TransactionCount,
		"isAdmin":          isAdmin == 1,
	}
	c.JSON(http.StatusOK, response)
}

// UpsertCurrentUser creates or updates the authenticated user's profile
func (h *HTTPHandler) UpsertCurrentUser(c *gin.Context) {
	uidValue, exists := c.Get("uid")
	if !exists || uidValue == "" {
		// Firebase未初期化の場合、テスト用に固定UIDを使用
		fmt.Println("[WARN] No uid in context for upsert - using test user")
		uidValue = "18oYncIdc3UuvZneYQQ4j2II23A2"
	}
	uid := uidValue.(string)

	var req struct {
		Name      string  `json:"name" binding:"required"`
		AvatarURL *string `json:"avatarUrl"`
		Bio       *string `json:"bio"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("[DEBUG] UpsertCurrentUser: uid=%s, name=%s\n", uid, req.Name)

	_, err := h.db.Exec(`
	   INSERT INTO users (id, username, avatar_url, bio)
	   VALUES (?, ?, ?, ?)
	   ON DUPLICATE KEY UPDATE
	       username = VALUES(username),
	       avatar_url = VALUES(avatar_url),
	       bio = VALUES(bio)
	`, uid, req.Name, req.AvatarURL, req.Bio)
	if err != nil {
		fmt.Printf("[ERROR] UpsertCurrentUser exec failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var u User
	var isAdmin int
	err = h.db.QueryRow("SELECT id, name, avatar_url, bio, rating, selling_count, follower_count, review_count, is_admin FROM users WHERE id = ?", uid).
		Scan(&u.ID, &u.Name, &u.AvatarURL, &u.Bio, &u.Rating, &u.SellingCount, &u.FollowerCount, &u.ReviewCount, &isAdmin)
	if err != nil {
		fmt.Printf("[ERROR] UpsertCurrentUser query failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("[DEBUG] UpsertCurrentUser success: uid=%s\n", uid)
	response := gin.H{
		"id":            u.ID,
		"name":          u.Name,
		"avatarUrl":     u.AvatarURL,
		"bio":           u.Bio,
		"rating":        u.Rating,
		"sellingCount":  u.SellingCount,
		"followerCount": u.FollowerCount,
		"reviewCount":   u.ReviewCount,
		"isAdmin":       isAdmin == 1,
	}
	c.JSON(http.StatusOK, response)
}

// GetUsers returns all users
func (h *HTTPHandler) GetUsers(c *gin.Context) {
		rows, err := h.db.Query("SELECT id, username, avatar_url, bio, rating, listings_count, follower_count, review_count, transaction_count FROM users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var userList []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.AvatarURL, &u.Bio, &u.Rating, &u.SellingCount, &u.FollowerCount, &u.ReviewCount, &u.TransactionCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		userList = append(userList, u)
	}

	c.JSON(http.StatusOK, userList)
}

// GetUserByID returns a user by ID
func (h *HTTPHandler) GetUserByID(c *gin.Context) {
	id := c.Param("id")
	var u User
		err := h.db.QueryRow("SELECT id, username, avatar_url, bio, rating, listings_count, follower_count, review_count, transaction_count FROM users WHERE id = ?", id).
		    Scan(&u.ID, &u.Name, &u.AvatarURL, &u.Bio, &u.Rating, &u.SellingCount, &u.FollowerCount, &u.ReviewCount, &u.TransactionCount)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 動的に出品数を再計算（データベースとの同期を確認）
	var actualSellingCount int
		err = h.db.QueryRow("SELECT COUNT(*) FROM items WHERE seller_id = ? AND is_sold_out = 0", id).Scan(&actualSellingCount)
		if err == nil {
		    // listings_countカラムに合わせて更新
		    if u.SellingCount != nil && actualSellingCount != *u.SellingCount {
			fmt.Printf("[SYNC] Updating listings_count for user %s: %d -> %d\n", id, *u.SellingCount, actualSellingCount)
			_, updateErr := h.db.Exec("UPDATE users SET listings_count = ? WHERE id = ?", actualSellingCount, id)
			if updateErr != nil {
			    fmt.Printf("[ERROR] Failed to sync listings_count: %v\n", updateErr)
			} else {
			    u.SellingCount = &actualSellingCount
			}
		    }
		}

	c.JSON(http.StatusOK, u)
}

// FollowUser follows a user
func (h *HTTPHandler) FollowUser(c *gin.Context) {
	var req struct {
		FollowerID string `json:"followerId" binding:"required"`
		FolloweeID string `json:"followeeId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.FollowerID == req.FolloweeID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot follow yourself"})
		return
	}

	followID := generateID()
	_, err := h.db.Exec("INSERT OR IGNORE INTO user_follows (id, follower_id, followee_id) VALUES (?, ?, ?)",
		followID, req.FollowerID, req.FolloweeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update follower count
	h.db.Exec("UPDATE users SET follower_count = (SELECT COUNT(*) FROM user_follows WHERE followee_id = ?) WHERE id = ?",
		req.FolloweeID, req.FolloweeID)

	var follow UserFollow
	err = h.db.QueryRow("SELECT id, follower_id, followee_id, created_at FROM user_follows WHERE follower_id = ? AND followee_id = ?",
		req.FollowerID, req.FolloweeID).
		Scan(&follow.ID, &follow.FollowerID, &follow.FolloweeID, &follow.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve follow record: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, follow)
}

// UnfollowUser unfollows a user
func (h *HTTPHandler) UnfollowUser(c *gin.Context) {
	var req struct {
		FollowerID string `json:"followerId" binding:"required"`
		FolloweeID string `json:"followeeId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.db.Exec("DELETE FROM user_follows WHERE follower_id = ? AND followee_id = ?",
		req.FollowerID, req.FolloweeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Follow relationship not found"})
		return
	}

	// Update follower count
	h.db.Exec("UPDATE users SET follower_count = (SELECT COUNT(*) FROM user_follows WHERE followee_id = ?) WHERE id = ?",
		req.FolloweeID, req.FolloweeID)

	c.JSON(http.StatusOK, gin.H{"message": "Unfollowed successfully"})
}

// GetFollowers returns followers of a user
func (h *HTTPHandler) GetFollowers(c *gin.Context) {
	userID := c.Param("userId")
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	rows, err := h.db.Query(`
		SELECT f.id, f.follower_id, f.followee_id, f.created_at
		FROM user_follows f
		WHERE f.followee_id = ?
		ORDER BY f.created_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var follows []UserFollow
	for rows.Next() {
		var f UserFollow
		if err := rows.Scan(&f.ID, &f.FollowerID, &f.FolloweeID, &f.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		follows = append(follows, f)
	}

	c.JSON(http.StatusOK, gin.H{"followers": follows})
}

// GetFollowing returns users being followed by a user
func (h *HTTPHandler) GetFollowing(c *gin.Context) {
	userID := c.Param("userId")
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	rows, err := h.db.Query(`
		SELECT f.id, f.follower_id, f.followee_id, f.created_at
		FROM user_follows f
		WHERE f.follower_id = ?
		ORDER BY f.created_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var follows []UserFollow
	for rows.Next() {
		var f UserFollow
		if err := rows.Scan(&f.ID, &f.FollowerID, &f.FolloweeID, &f.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		follows = append(follows, f)
	}

	c.JSON(http.StatusOK, gin.H{"following": follows})
}

// CreateReview creates a review for a user
func (h *HTTPHandler) CreateReview(c *gin.Context) {
	var req struct {
		ReviewerID string  `json:"reviewerId" binding:"required"`
		RevieweeID string  `json:"revieweeId" binding:"required"`
		Rating     float64 `json:"rating" binding:"required,min=1,max=5"`
		Comment    string  `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ReviewerID == req.RevieweeID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot review yourself"})
		return
	}

	// 取引実績のある買い手のみが売り手を評価可能
	var txCount int
	err := h.db.QueryRow(`
		SELECT COUNT(*) FROM transactions 
		WHERE buyer_id = ? AND seller_id = ?
	`, req.ReviewerID, req.RevieweeID).Scan(&txCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if txCount == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "No trade history between reviewer and reviewee"})
		return
	}

	reviewID := generateID()
	_, err = h.db.Exec("INSERT INTO user_reviews (id, reviewer_id, reviewee_id, rating, comment) VALUES (?, ?, ?, ?, ?)",
		reviewID, req.ReviewerID, req.RevieweeID, req.Rating, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update user rating and review count
	h.db.Exec(`
		UPDATE users SET 
			rating = (SELECT AVG(rating) FROM user_reviews WHERE reviewee_id = ?),
			review_count = (SELECT COUNT(*) FROM user_reviews WHERE reviewee_id = ?)
		WHERE id = ?
	`, req.RevieweeID, req.RevieweeID, req.RevieweeID)

	var review UserReview
	h.db.QueryRow("SELECT id, reviewer_id, reviewee_id, rating, comment, created_at FROM user_reviews WHERE id = ?", reviewID).
		Scan(&review.ID, &review.ReviewerID, &review.RevieweeID, &review.Rating, &review.Comment, &review.CreatedAt)

	c.JSON(http.StatusCreated, review)
}

// GetUserReviews returns reviews for a user
func (h *HTTPHandler) GetUserReviews(c *gin.Context) {
	userID := c.Param("userId")

	rows, err := h.db.Query(`
		SELECT r.id, r.reviewer_id, r.reviewee_id, r.rating, r.comment, r.created_at
		FROM user_reviews r
		WHERE r.reviewee_id = ?
		ORDER BY r.created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var reviews []UserReview
	for rows.Next() {
		var r UserReview
		if err := rows.Scan(&r.ID, &r.ReviewerID, &r.RevieweeID, &r.Rating, &r.Comment, &r.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		reviews = append(reviews, r)
	}

	c.JSON(http.StatusOK, reviews)
}

// SetDBAdmin updates users.is_admin flag in the database
func (h *HTTPHandler) SetDBAdmin(c *gin.Context) {
	var req struct {
		UserID  string `json:"userId" binding:"required"`
		IsAdmin bool   `json:"isAdmin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	val := 0
	if req.IsAdmin {
		val = 1
	}

	res, err := h.db.Exec("UPDATE users SET is_admin = ? WHERE id = ?", val, req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"userId": req.UserID, "isAdmin": req.IsAdmin})
}

// SearchUsersAndItems searches both users and items by name/title with optional category filter
func (h *HTTPHandler) SearchUsersAndItems(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	category := c.Query("category")
	pattern := "%" + query + "%"

	// Search users
	userRows, err := h.db.Query(`
		SELECT id, name, avatar_url, rating
		FROM users
		WHERE name LIKE ?
		LIMIT 20
	`, pattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer userRows.Close()

	var users []map[string]interface{}
	for userRows.Next() {
		var id, name string
		var avatarURL *string
		var rating *float64
		if err := userRows.Scan(&id, &name, &avatarURL, &rating); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		users = append(users, map[string]interface{}{
			"id":        id,
			"name":      name,
			"avatarUrl": avatarURL,
			"rating":    rating,
		})
	}

	// Search items with optional category filter
	var itemRows *sql.Rows
	if category != "" {
		itemRows, err = h.db.Query(`
			SELECT id, title, price, image_url, seller_id
			FROM items
			WHERE (title LIKE ? OR description LIKE ?) AND category = ?
			LIMIT 20
		`, pattern, pattern, category)
	} else {
		itemRows, err = h.db.Query(`
			SELECT id, title, price, image_url, seller_id
			FROM items
			WHERE title LIKE ? OR description LIKE ?
			LIMIT 20
		`, pattern, pattern)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer itemRows.Close()

	var items []map[string]interface{}
	for itemRows.Next() {
		var id, title, imageURL string
		var price int
		var sellerID *string
		if err := itemRows.Scan(&id, &title, &price, &imageURL, &sellerID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, map[string]interface{}{
			"id":       id,
			"title":    title,
			"price":    price,
			"imageUrl": imageURL,
			"sellerId": sellerID,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"items": items,
	})
}
