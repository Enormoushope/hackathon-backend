package handlers

import (
	"backend/internal/db"
	"backend/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 単一ユーザー情報取得API (/api/users/:uid)
func GetUserByID(c *gin.Context) {
	userID := c.Param("uid")
	var user models.User
	err := db.DB.QueryRow("SELECT id, name, email, avatar_url, created_at FROM users WHERE id = ?", userID).
		Scan(&user.ID, &user.Name, &user.Email, &user.AvatarURL, &user.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func GetUserProfile(c *gin.Context) {
	userID := c.Param("uid") // Firebaseから送られてくるUID

	var profile models.UserProfileResponse

	// 1. ユーザー基本情報
	db.DB.QueryRow("SELECT id, name, email, avatar_url FROM users WHERE id = ?", userID).
		Scan(&profile.User.ID, &profile.User.Name, &profile.User.Email, &profile.User.AvatarURL)

	// 2. 出品中の商品（これに紐づくDMもフロントでフィルタリングできるよう商品IDを付与）
	rows, _ := db.DB.Query("SELECT id, title, price, image_url FROM products WHERE seller_id = ?", userID)
	for rows.Next() {
		var p models.Product
		rows.Scan(&p.ID, &p.Title, &p.Price, &p.ImageURL)
		profile.SellingProducts = append(profile.SellingProducts, p)
	}

	// 3. いいねした商品
	likedRows, _ := db.DB.Query(`
		SELECT p.id, p.title, p.price, p.image_url 
		FROM products p JOIN likes l ON p.id = l.product_id 
		WHERE l.user_id = ?`, userID)
	for likedRows.Next() {
		var p models.Product
		likedRows.Scan(&p.ID, &p.Title, &p.Price, &p.ImageURL)
		profile.LikedProducts = append(profile.LikedProducts, p)
	}

	// 4. DM履歴（自分が関わっている全てのメッセージ）
	msgRows, _ := db.DB.Query("SELECT product_id, sender_id, content FROM messages WHERE sender_id = ? OR receiver_id = ?", userID, userID)
	for msgRows.Next() {
		var m models.Message
		msgRows.Scan(&m.ProductID, &m.SenderID, &m.Content)
		profile.LatestMessages = append(profile.LatestMessages, m)
	}

	c.JSON(http.StatusOK, profile)
}

func SyncUser(c *gin.Context) {
    var u models.User
    if err := c.ShouldBindJSON(&u); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // ON DUPLICATE KEY UPDATE を使って、存在しなければ作成、あれば更新
    _, err := db.DB.Exec(
        "INSERT INTO users (id, name, email, avatar_url) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE name=?, avatar_url=?",
        u.ID, u.Name, u.Email, u.AvatarURL, u.Name, u.AvatarURL,
    )
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "ユーザー同期に失敗しました"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "user synced"})
}