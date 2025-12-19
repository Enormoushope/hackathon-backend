package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/cors"
)

func main() {
	// 環境変数の確認用デバッグログ
	log.Printf("★DEBUG CHECK: DB_HOST=[%s] INSTANCE_CONNECTION_NAME=[%s]", os.Getenv("DB_HOST"), os.Getenv("INSTANCE_CONNECTION_NAME"))
	// ==========================================
	// 1. データベース接続設定 (ここが最重要)
	// ==========================================
	dbUser := os.Getenv("MYSQL_USER")
	dbPass := os.Getenv("MYSQL_PASSWORD")
	dbName := os.Getenv("MYSQL_DATABASE")
	instanceConnectionName := os.Getenv("INSTANCE_CONNECTION_NAME") // Cloud Run用の変数

	var dsn string
	if instanceConnectionName != "" {
		// Cloud Run用: Unixソケット
		protocol := "unix"
		address := fmt.Sprintf("/cloudsql/%s", instanceConnectionName)
		log.Println("Connecting via Unix Socket...")
		dsn = fmt.Sprintf("%s:%s@%s(%s)/%s?parseTime=true&loc=Local", dbUser, dbPass, protocol, address, dbName)
	} else {
		// ローカル用: TCP
		protocol := "tcp"
		dbHost := os.Getenv("MYSQL_HOST")
		if dbHost == "" {
			dbHost = "127.0.0.1"
		}
		dbPort := os.Getenv("DB_PORT")
		if dbPort == "" {
			dbPort = "3306"
		}
		address := fmt.Sprintf("%s:%s", dbHost, dbPort)
		log.Println("Connecting via TCP (Local)...")
		dsn = fmt.Sprintf("%s:%s@%s(%s)/%s?parseTime=true&loc=Local", dbUser, dbPass, protocol, address, dbName)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("DB Open Error: %v", err)
	}
	defer db.Close()

	// 接続確認 (ここで失敗するとCloud Runは起動失敗とみなす)
	if err := db.Ping(); err != nil {
		log.Fatalf("DB Ping Error: %v", err)
	}
	log.Println("Successfully connected to the database!")

	// ==========================================
	// 2. ルーティング設定 (あなたの既存の処理)
	// ==========================================
	mux := http.NewServeMux()

	// ★ここにあなたのハンドラを追加してください
	// 例: mux.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) { ... })

	// 単純なヘルスチェック用エンドポイント（デバッグ用）
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from Cloud Run! DB Connected."))
	})

	// ==========================================
	// 3. CORS設定 (フロントエンドからの接続許可)
	// ==========================================
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "PUT", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	})
	// ==========================================
	// 4. サーバー起動 (ここも重要)
	// ==========================================
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // デフォルト
	}

	log.Printf("Server listening on port %s", port)
	// muxをCORSミドルウェアで包む
	if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
// CORS設定用のミドルウェア関数
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// どのドメインからのアクセスも許可する
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// 許可するメソッド
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		// 許可するヘッダー
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// ブラウザからの事前確認(OPTIONS)にはOKだけ返す
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
}

// DB接続関数: 環境変数によってUnixソケットとTCPを切り替え
func connectDB() (*sql.DB, bool, error) {
	dbUser := os.Getenv("MYSQL_USER")
	dbPass := os.Getenv("MYSQL_PASSWORD")
	dbName := os.Getenv("MYSQL_DATABASE")
	instanceConnectionName := os.Getenv("INSTANCE_CONNECTION_NAME")

	var dsn string
	if instanceConnectionName != "" {
		// Cloud Run/本番用（Unix Socket）
		dsn = fmt.Sprintf("%s:%s@unix(/cloudsql/%s)/%s?parseTime=true&loc=Local", dbUser, dbPass, instanceConnectionName, dbName)
		log.Printf("Connecting to MySQL via Unix socket: /cloudsql/%s (db=%s)\n", instanceConnectionName, dbName)
		db, err := sql.Open("mysql", dsn)
		return db, true, err
	} else {
		// ローカル用（TCP）
		dbHost := os.Getenv("DB_HOST")
		dbPort := os.Getenv("DB_PORT")
		if dbHost == "" {
			dbHost = "127.0.0.1"
		}
		if dbPort == "" {
			dbPort = "3306"
		}
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local", dbUser, dbPass, dbHost, dbPort, dbName)
		log.Printf("Connecting to MySQL via TCP: %s:%s (db=%s)\n", dbHost, dbPort, dbName)
		db, err := sql.Open("mysql", dsn)
		return db, false, err
	}
}
