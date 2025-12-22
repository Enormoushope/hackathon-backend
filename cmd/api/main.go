package main

import (
	"backend/internal/db"
	"backend/internal/handlers"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. データベースの初期化
	db.InitDB()

	// 2. Ginルーターの初期化
	r := gin.Default()

	// 3. CORSの設定 (Vercelとローカル両方を許可)
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://localhost:5173",
		"https://hackathon-frontend-jet.vercel.app",
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	// 4. APIルートのグループ化
	api := r.Group("/api")
	{
		// --- ヘルスチェック ---
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		// --- 商品関連 (Products) ---
		api.GET("/products", handlers.GetProducts)
		api.GET("/products/:id", handlers.GetProductByID)
		api.POST("/products", handlers.CreateProduct)
		api.POST("/products/:id/purchase", handlers.PurchaseProduct) // 購入処理

		// --- ユーザー関連 ---
		api.GET("/users/:uid", handlers.GetUserByID)
		api.GET("/users/:uid/profile", handlers.GetUserProfile)
		api.POST("/users/sync", handlers.SyncUser)

		// --- いいね・DM関連 ---
		api.POST("/likes/toggle", handlers.ToggleLike)
		api.GET("/likes/status", handlers.CheckLikeStatus)
		api.POST("/messages", handlers.SendMessage)
		api.GET("/messages", handlers.GetChatHistory)

		// --- Gemini AI連携関連 (ここをReactのURLに合わせる) ---
		// Reactの Sell.tsx が axios.post("/api/ai/description") を叩くので合わせます
		api.POST("/ai/description", handlers.GenerateAIDescription)
		api.POST("/ai/suggest-price", handlers.SuggestAIPrice)

		api.GET("/debug-env", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"project_id": os.Getenv("GCP_PROJECT_ID"),
				"port":       os.Getenv("PORT"),
				"instance":   os.Getenv("INSTANCE_CONNECTION_NAME"),
			})
		})
	}

	// 5. サーバー起動
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server is running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("サーバーの起動に失敗しました:", err)
	}
}
