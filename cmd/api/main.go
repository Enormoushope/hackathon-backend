package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"hackathon/backend/internal/interface/http"

	firebase "firebase.google.com/go/v4"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 1. サーバーポート設定
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 2. 環境変数の取得
	dbUser := os.Getenv("MYSQL_USER")
	dbPass := os.Getenv("MYSQL_PASSWORD")
	dbName := os.Getenv("MYSQL_DATABASE")
	instanceConnectionName := os.Getenv("INSTANCE_CONNECTION_NAME")

	var dsn string
	if instanceConnectionName != "" {
		// 【重要】Cloud Run用: Unixソケット形式
		// dial tcp が出ないように、しっかり @unix() 形式で記述
		dsn = fmt.Sprintf("%s:%s@unix(/cloudsql/%s)/%s?parseTime=true&loc=Local",
			dbUser, dbPass, instanceConnectionName, dbName)
		log.Printf("Connecting to Cloud SQL via Unix Socket: %s", instanceConnectionName)
	} else {
		// ローカル用: TCP形式
		dbHost := os.Getenv("MYSQL_HOST")
		if dbHost == "" {
			dbHost = "127.0.0.1"
		}
		dbPort := os.Getenv("DB_PORT")
		if dbPort == "" {
			dbPort = "3306"
		}
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local",
			dbUser, dbPass, dbHost, dbPort, dbName)
		log.Printf("Connecting to DB via TCP: %s:%s", dbHost, dbPort)
	}

	// 3. データベースオープン
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("⚠️ DB Open Error: %v", err)
	}
	defer db.Close()

	// 接続テスト (Ping)
	if err := db.Ping(); err != nil {
		log.Printf("⚠️ WARNING: データベース接続に失敗しました: %v", err)
	} else {
		log.Println("✅ Successfully connected to the database!")
	}

	// Firebase初期化
	ctx := context.Background()
	firebaseApp, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Fatalf("Firebase app initialization failed: %v", err)
	}
	authClient, err := firebaseApp.Auth(ctx)
	if err != nil {
		log.Fatalf("Firebase auth client initialization failed: %v", err)
	}
	firebaseAuthManager := http.NewFirebaseAuthManager(authClient)

	// 4. Gin ルーター設定
	router := gin.Default()

	// CORS ミドルウェア
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	// HTTPHandler のインスタンス化
	handler := http.NewHTTPHandler(db, firebaseAuthManager)

	// ルート設定
	api := router.Group("/api")
	{
		// Auth routes
		api.POST("/auth/me", handler.UpsertCurrentUser)
		api.GET("/auth/me", handler.GetCurrentUser)

		// User routes
		api.GET("/users", handler.GetUsers)
		api.GET("/users/:id", handler.GetUserByID)
		api.POST("/users/:id/follow", handler.FollowUser)
		api.DELETE("/users/:id/follow", handler.UnfollowUser)
		api.GET("/users/:userId/followers", handler.GetFollowers)
		api.GET("/users/:userId/following", handler.GetFollowing)
		api.POST("/users/:userId/reviews", handler.CreateReview)
		api.GET("/users/:userId/reviews", handler.GetUserReviews)

		// Item routes
		api.GET("/items", handler.GetItems)
		api.POST("/items", handler.CreateItem)
		api.GET("/items/:id", handler.GetItemByID)
		api.PUT("/items/:id", handler.UpdateItem)
		api.DELETE("/items/:id", handler.DeleteItem)
		api.POST("/items/:id/like", handler.LikeItem)
		api.DELETE("/items/:id/like", handler.UnlikeItem)
		api.GET("/items/:id/likes", handler.GetItemLikes)
		api.POST("/items/:id/comments", handler.CreateComment)
		api.GET("/items/:id/comments", handler.GetItemComments)
		api.DELETE("/items/:id/comments/:commentId", handler.DeleteComment)

		// Category routes
		api.GET("/categories", handler.GetCategories)

		// Transaction routes
		api.POST("/transactions", handler.CreateTransaction)
		api.GET("/transactions", handler.GetTransactions)
		api.GET("/transactions/:id", handler.GetTransactionByID)
		api.PUT("/transactions/:id", handler.UpdateTransaction)

		// Admin routes
		api.POST("/admin/set-admin", handler.SetDBAdmin)

		// Search routes
		api.GET("/search", handler.SearchUsersAndItems)
	}

	// 5. サーバー起動
	log.Printf("Server listening on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
