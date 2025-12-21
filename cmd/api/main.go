package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	stdhttp "net/http"
	"os"

	http "github.com/xyz77/hackathon/backend/internal/interface/http"

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
			c.AbortWithStatus(stdhttp.StatusOK)
			return
		}
		c.Next()
	})

	// HTTPHandler のインスタンス化
	handler := http.NewHTTPHandler(db, firebaseAuthManager)

	// ルート設定
	handler.RegisterRoutes(router)

	// 5. サーバー起動
	log.Printf("Server listening on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
