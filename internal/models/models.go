package models

import "time"

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

type Product struct {
	ID          int       `json:"id"`
	SellerID    string    `json:"seller_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       int       `json:"price"`
	ImageURL    string    `json:"image_url"`
	IsSold      bool      `json:"is_sold"` // 追加
	CreatedAt   time.Time `json:"created_at"`
	LikeCount   int       `json:"like_count"` // 追加: いいね数
}

type Message struct {
	ID         int       `json:"id"`
	ProductID  int       `json:"product_id"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

type Like struct {
	UserID    string `json:"user_id"`
	ProductID int    `json:"product_id"`
}

type UserProfileResponse struct {
	User            User      `json:"user"`
	SellingProducts []Product `json:"selling_products"`
	LikedProducts   []Product `json:"liked_products"`
	LatestMessages  []Message `json:"latest_messages"`
}
