package main

import (
	"log"
	"os"

	"backend/internal/db"
	"backend/internal/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. データベースの初期化 (テーブル作成含む)
	db.InitDB()

	// 2. Ginルーターの初期化
	r := gin.Default()

	// 3. CORSの設定 (Vercelからのアクセスを許可するために必須)
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
            "http://localhost:5173",                     // ローカル開発用
            "https://hackathon-frontend-jet.vercel.app", // ←これをついか！
        }// 本番環境では特定のドメインに絞ることを推奨
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
		api.GET("/products", handlers.GetProducts)            // 一覧取得
		api.GET("/products/:id", handlers.GetProductByID)    // 詳細取得
		api.POST("/products", handlers.CreateProduct)         // 出品

		// --- ユーザー・プロフィール関連 (User) ---
		api.GET("/users/:uid/profile", handlers.GetUserProfile) // プロフィール統合データ

		// --- いいね関連 (Likes) ---
		api.POST("/likes/toggle", handlers.ToggleLike)        // いいね登録/解除
		api.GET("/likes/status", handlers.CheckLikeStatus)    // いいね状態確認

		// --- DM関連 (Messages) ---
		api.POST("/messages", handlers.SendMessage)           // メッセージ送信
		api.GET("/messages", handlers.GetChatHistory)         // チャット履歴取得

		// --- Gemini AI連携関連 ---
		api.POST("/ai/describe", handlers.GenerateAIDescription) // 商品説明の自動生成
		api.POST("/ai/suggest-price", handlers.SuggestAIPrice)    // 適正価格の査定

		api.POST("/users/sync", handlers.SyncUser) // ユーザー情報の同期
	}

	// 5. ポート設定とサーバー起動
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // ローカル実行時のデフォルト
	}

	log.Printf("🚀 Server is running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("サーバーの起動に失敗しました:", err)
	}
}