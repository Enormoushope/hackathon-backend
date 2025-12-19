package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 環境変数の確認用デバッグログ
	log.Printf("★DEBUG CHECK: DB_HOST=[%s] INSTANCE_CONNECTION_NAME=[%s]", os.Getenv("DB_HOST"), os.Getenv("INSTANCE_CONNECTION_NAME"))

	// 1. データベース接続設定
	dbUser := os.Getenv("MYSQL_USER")
	dbPass := os.Getenv("MYSQL_PASSWORD")
	dbName := os.Getenv("MYSQL_DATABASE")
	instanceConnectionName := os.Getenv("INSTANCE_CONNECTION_NAME")

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

	// 接続確認
	if err := db.Ping(); err != nil {
		log.Fatalf("DB Ping Error: %v", err)
	}
	log.Println("Successfully connected to the database!")

	// 2. ルーティング設定
	mux := http.NewServeMux()
	// ヘルスチェック用
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from Cloud Run! DB Connected."))
	})
	// ★ここにAPIの処理を追加する場合は mux.HandleFunc を使う

	// 3. サーバー起動
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server listening on port %s", port)
	// muxをCORSミドルウェアで包んで起動
	if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// ==========================================
// 4. CORS設定 (main関数の外に配置！)
// ==========================================
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
