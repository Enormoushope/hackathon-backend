package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetUserReports - 通報一覧を取得 (管理者のみ)
func (h *HTTPHandler) GetUserReports(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, reporter_id, reported_user_id, reason, description, status, created_at, updated_at
		FROM user_reports
		ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var reports []UserReport
	for rows.Next() {
		var report UserReport
		if err := rows.Scan(&report.ID, &report.ReporterID, &report.ReportedUserID, &report.Reason, &report.Description, &report.Status, &report.CreatedAt, &report.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		reports = append(reports, report)
	}

	c.JSON(http.StatusOK, reports)
}

// ReportUser - ユーザーを通報
type ReportUserRequest struct {
	ReportedUserID string `json:"reportedUserId" binding:"required"`
	Reason         string `json:"reason" binding:"required"`
	Description    string `json:"description"`
}

func (h *HTTPHandler) ReportUser(c *gin.Context) {
	var req ReportUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var reporterID string
	if uidVal, exists := c.Get("uid"); exists {
		if s, ok := uidVal.(string); ok && s != "" {
			reporterID = s
		}
	}
	if reporterID == "" {
		reporterID = "18oYncIdc3UuvZneYQQ4j2II23A2"
	}

	id := generateID()
	_, err := h.db.Exec(`
		INSERT INTO user_reports (id, reporter_id, reported_user_id, reason, description, status)
		VALUES (?, ?, ?, ?, ?, 'pending')
	`, id, reporterID, req.ReportedUserID, req.Reason, req.Description)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

// GetAllUsersAdmin - 全ユーザーを取得 (管理者用)
func (h *HTTPHandler) GetAllUsersAdmin(c *gin.Context) {
	status := c.Query("status") // pending, approved, rejected

	query := `
		SELECT u.id, u.name, u.avatar_url, u.bio, u.rating, u.listings_count, u.follower_count, u.review_count, u.transaction_count,
			   COUNT(r.id) as report_count
		FROM users u
		LEFT JOIN user_reports r ON u.id = r.reported_user_id AND r.status = 'pending'
		GROUP BY u.id
	`

	if status == "reported" {
		query += " HAVING COUNT(r.id) > 0"
	}

	query += " ORDER BY u.id DESC"

	rows, err := h.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type UserAdmin struct {
		ID               string   `json:"id"`
		Name             string   `json:"name"`
		AvatarURL        *string  `json:"avatarUrl"`
		Bio              *string  `json:"bio"`
		Rating           *float64 `json:"rating"`
		SellingCount     *int     `json:"sellingCount"`
		FollowerCount    *int     `json:"followerCount"`
		ReviewCount      *int     `json:"reviewCount"`
		TransactionCount *int     `json:"transactionCount"`
		ReportCount      int      `json:"reportCount"`
	}

	var users []UserAdmin
	for rows.Next() {
		var u UserAdmin
		if err := rows.Scan(&u.ID, &u.Name, &u.AvatarURL, &u.Bio, &u.Rating, &u.SellingCount, &u.FollowerCount, &u.ReviewCount, &u.TransactionCount, &u.ReportCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		users = append(users, u)
	}

	c.JSON(http.StatusOK, users)
}

// GetUserDetailsAdmin - ユーザーの詳細情報を取得 (管理者用)
func (h *HTTPHandler) GetUserDetailsAdmin(c *gin.Context) {
	userID := c.Param("id")

	// ユーザー情報
	var user struct {
		ID               string
		Name             string
		AvatarURL        *string
		Bio              *string
		Rating           *float64
		SellingCount     *int
		FollowerCount    *int
		ReviewCount      *int
		TransactionCount *int
		IsAdmin          bool
	}

	err := h.db.QueryRow(`
		SELECT id, name, avatar_url, bio, rating, listings_count, follower_count, review_count, transaction_count, is_admin
		FROM users
		WHERE id = ?
	`, userID).Scan(&user.ID, &user.Name, &user.AvatarURL, &user.Bio, &user.Rating, &user.SellingCount, &user.FollowerCount, &user.ReviewCount, &user.TransactionCount, &user.IsAdmin)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 通報履歴
	reportRows, err := h.db.Query(`
		SELECT id, reporter_id, reason, description, status, created_at
		FROM user_reports
		WHERE reported_user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer reportRows.Close()

	var reports []UserReport
	for reportRows.Next() {
		var report UserReport
		if err := reportRows.Scan(&report.ID, &report.ReporterID, &report.Reason, &report.Description, &report.Status, &report.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		reports = append(reports, report)
	}

	// 取引履歴
	tradeRows, err := h.db.Query(`
		SELECT id, item_id, price, seller_id, created_at
		FROM transactions
		WHERE seller_id = ? OR buyer_id = ?
		ORDER BY created_at DESC
	`, userID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tradeRows.Close()

	var trades []gin.H
	for tradeRows.Next() {
		var id, itemID string
		var price int
		var sellerID *string
		var createdAt string
		if err := tradeRows.Scan(&id, &itemID, &price, &sellerID, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		trades = append(trades, gin.H{
			"id":        id,
			"itemId":    itemID,
			"price":     price,
			"sellerId":  sellerID,
			"createdAt": createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"user":    user,
		"reports": reports,
		"trades":  trades,
	})
}
