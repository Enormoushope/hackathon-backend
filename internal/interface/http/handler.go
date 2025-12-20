package http

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// CategoryNode mirrors frontend schema
type CategoryNode struct {
	Code     string         `json:"code"`
	Label    string         `json:"label"`
	Children []CategoryNode `json:"children,omitempty"`
}

// DefaultClassificationTree returns a minimal seed; can be extended
func DefaultClassificationTree() []CategoryNode {
	return []CategoryNode{
		{
			Code:  "000",
			Label: "資産・投資 (Asset & Investment)",
			Children: []CategoryNode{
				{Code: "010", Label: "Trading Cards (トレカ)"},
				{Code: "020", Label: "Graded Slabs (鑑定品)"},
			},
		},
		{
			Code:  "100",
			Label: "本 (Books)",
			Children: []CategoryNode{
				{Code: "110", Label: "漫画・コミック"},
				{Code: "120", Label: "ビジネス・実用"},
			},
		},
	}
}

// User model
type User struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	AvatarURL        *string  `json:"avatarUrl,omitempty"`
	Bio              *string  `json:"bio,omitempty"`
	Rating           *float64 `json:"rating,omitempty"`
	SellingCount     *int     `json:"sellingCount,omitempty"`
	FollowerCount    *int     `json:"followerCount,omitempty"`
	ReviewCount      *int     `json:"reviewCount,omitempty"`
	TransactionCount *int     `json:"transactionCount,omitempty"`
}

// Item model
type Item struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Price        int      `json:"price"`
	Description  *string  `json:"description,omitempty"`
	Condition    *string  `json:"condition,omitempty"`
	Category     *string  `json:"category,omitempty"`
	ImageURL     string   `json:"image_Url"`
	IsSoldOut    bool     `json:"is_SoldOut"`
	SellerID     *string  `json:"seller_Id,omitempty"`
	IsInvestItem *bool    `json:"isInvestItem,omitempty"`
	ViewCount    *int     `json:"viewCount,omitempty"`
	LikeCount    *int     `json:"likeCount,omitempty"`
	WatchCount   *int     `json:"watchCount,omitempty"`
	SellerRating *float64 `json:"sellerRating,omitempty"`
	ProductGroup *string  `json:"productGroup,omitempty"`
}

// Conversation model
type Conversation struct {
	ID        string `json:"id"`
	ItemID    string `json:"itemId"`
	BuyerID   string `json:"buyerId"`
	SellerID  string `json:"sellerId"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// Message model
type Message struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversationId"`
	SenderID       string `json:"senderId"`
	Content        string `json:"content"`
	CreatedAt      string `json:"createdAt"`
}

// ItemReaction model
type ItemReaction struct {
	ID           string `json:"id"`
	ItemID       string `json:"itemId"`
	UserID       string `json:"userId"`
	ReactionType string `json:"reactionType"`
	CreatedAt    string `json:"createdAt"`
}

// UserFollow model
type UserFollow struct {
	ID         string `json:"id"`
	FollowerID string `json:"followerId"`
	FolloweeID string `json:"followeeId"`
	CreatedAt  string `json:"createdAt"`
}

// UserReport model
type UserReport struct {
	ID             string `json:"id"`
	ReporterID     string `json:"reporterId"`
	ReportedUserID string `json:"reportedUserId"`
	Reason         string `json:"reason"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

// UserReview model
type UserReview struct {
	ID         string  `json:"id"`
	ReviewerID string  `json:"reviewerId"`
	RevieweeID string  `json:"revieweeId"`
	Rating     float64 `json:"rating"`
	Comment    string  `json:"comment"`
	CreatedAt  string  `json:"createdAt"`
}

// HTTPHandler represents the HTTP server
type HTTPHandler struct {
	db                  *sql.DB
	firebaseAuthManager *FirebaseAuthManager
	vertexAIManager     *VertexAIManager
	categoryMaster      []CategoryNode
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(db *sql.DB, firebaseAuthManager *FirebaseAuthManager) *HTTPHandler {
	return &HTTPHandler{
		db:                  db,
		firebaseAuthManager: firebaseAuthManager,
		categoryMaster:      DefaultClassificationTree(),
	}
}

// SetVertexAIManager sets the VertexAI manager
func (h *HTTPHandler) SetVertexAIManager(manager *VertexAIManager) {
	h.vertexAIManager = manager
}

// RegisterRoutes registers all HTTP routes
func (h *HTTPHandler) RegisterRoutes(router *gin.Engine) {
	// Debug endpoint without middleware
	router.GET("/health", func(c *gin.Context) {
		status := "OK"
		if h.firebaseAuthManager == nil {
			status = "Firebase not initialized"
		}
		c.JSON(200, gin.H{"status": status})
	})

	// Public API group (no auth required by default)
	api := router.Group("/api")
	// Attach token verification middleware so admin checks can read UID/claims when tokens are present
	api.Use(VerifyTokenMiddleware(h.firebaseAuthManager))
	{
		// Category master routes
		api.GET("/categories", h.GetCategories)
		api.PUT("/categories", AdminCheckMiddlewareWithDB(h.firebaseAuthManager, h.db), h.PutCategories)
		// User routes
		api.GET("/users", h.GetUsers)
		api.GET("/users/:id", h.GetUserByID)
		api.GET("/auth/me", h.GetCurrentUser)
		api.POST("/auth/me", h.UpsertCurrentUser)

		// Item routes
		api.GET("/items", h.GetItems)
		api.GET("/items/:id", h.GetItemByID)
		api.POST("/items/:id/increment-view", h.IncrementViewCount)
		api.POST("/items", h.CreateListing)

		// Investment routes
		api.POST("/investment-assets", h.CreateInvestmentAsset)
		api.GET("/investment-assets/:itemId", h.GetInvestmentAsset)

		// Warehouse storage routes
		api.POST("/warehouse-storage", h.CreateWarehouseStorage)
		api.GET("/warehouse-storage/:itemId", h.GetWarehouseStorage)

		// Chat routes
		api.POST("/conversations", h.CreateConversation)
		api.GET("/conversations/:id", h.GetConversation)
		api.GET("/conversations", h.GetConversationsByUser)
		api.POST("/messages", h.SendMessage)
		api.GET("/conversations/:id/messages", h.GetMessages)

		// Reaction routes
		api.POST("/reactions", h.AddReaction)
		api.DELETE("/reactions", h.RemoveReaction)
		api.GET("/reactions/items/:itemId", h.GetItemReactions)
		api.GET("/reactions/users/:userId", h.GetUserReactions)

		// Follow routes
		api.POST("/follows", h.FollowUser)
		api.DELETE("/follows", h.UnfollowUser)
		api.GET("/follows/followers/:userId", h.GetFollowers)
		api.GET("/follows/following/:userId", h.GetFollowing)

		// Review routes
		api.POST("/reviews", h.CreateReview)
		api.GET("/reviews/user/:userId", h.GetUserReviews)

		// Report routes (public submission)
		api.POST("/reports", h.ReportUser)

		// Transaction routes
		api.POST("/transactions", h.CreateTransaction)
		api.GET("/transactions/user/:userId", h.GetUserTransactions)
		api.GET("/transactions", AdminCheckMiddlewareWithDB(h.firebaseAuthManager, h.db), h.GetAllTransactions)
		api.POST("/transactions/complete", h.CompletePurchase)

		// Price history routes
		api.GET("/price-history/:itemId", h.GetPriceHistory)

		// Admin routes
		api.GET("/admin/reports", AdminCheckMiddlewareWithDB(h.firebaseAuthManager, h.db), h.GetUserReports)

		// Search routes
		api.GET("/search", h.SearchUsersAndItems)
		api.POST("/admin/reports", AdminCheckMiddlewareWithDB(h.firebaseAuthManager, h.db), h.ReportUser)
		api.GET("/admin/users", AdminCheckMiddlewareWithDB(h.firebaseAuthManager, h.db), h.GetAllUsersAdmin)
		api.GET("/admin/users/:id", AdminCheckMiddlewareWithDB(h.firebaseAuthManager, h.db), h.GetUserDetailsAdmin)
		api.POST("/admin/set-admin", AdminCheckMiddlewareWithDB(h.firebaseAuthManager, h.db), SetUserAdmin(h.firebaseAuthManager))
		// DB上の is_admin を更新する簡易管理API（開発ブートストラップ用）
		api.POST("/admin/db-set-admin", AdminCheckMiddlewareWithDB(h.firebaseAuthManager, h.db), h.SetDBAdmin)

		// AI routes
		api.POST("/ai/analyze", h.AnalyzeImage)
		api.POST("/ai/suggest-price", h.SuggestPrice)
		api.POST("/ai/suggest-description", h.SuggestDescription)
		api.POST("/ai/risk-assessment", h.RiskAssessment)
	}
}

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// GetCategories returns the current category master
func (h *HTTPHandler) GetCategories(c *gin.Context) {
	c.JSON(200, gin.H{"categories": h.categoryMaster})
}

// PutCategories replaces the category master with provided tree
func (h *HTTPHandler) PutCategories(c *gin.Context) {
	var payload struct {
		Categories []CategoryNode `json:"categories"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if len(payload.Categories) == 0 {
		c.JSON(400, gin.H{"error": "categories must not be empty"})
		return
	}
	h.categoryMaster = payload.Categories
	c.JSON(200, gin.H{"success": true})
}
