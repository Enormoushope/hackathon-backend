package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 1. サーバーポート設定
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 2. データベース接続設定
	dbUser := os.Getenv("MYSQL_USER")
	dbPass := os.Getenv("MYSQL_PASSWORD")
	dbName := os.Getenv("MYSQL_DATABASE")
	instanceConnectionName := os.Getenv("INSTANCE_CONNECTION_NAME")

	log.Printf("★Config: User=%s, DB=%s, Instance=%s", dbUser, dbName, instanceConnectionName)

	var dsn string
	if instanceConnectionName != "" {
		// Cloud Run用: 正しいUnixソケットの形式
		// [ユーザー名]:[パスワード]@unix(/cloudsql/[接続名])/[DB名]...
		dsn = fmt.Sprintf("%s:%s@unix(/cloudsql/%s)/%s?parseTime=true&loc=Local", dbUser, dbPass, instanceConnectionName, dbName)
		log.Println("Connecting via Unix Socket...")
	} else {
		// ローカル用: 正しいTCPの形式
		// [ユーザー名]:[パスワード]@tcp([ホスト]:[ポート])/[DB名]...
		dbHost := os.Getenv("MYSQL_HOST")
		if dbHost == "" {
			dbHost = "127.0.0.1"
		}
		dbPort := os.Getenv("DB_PORT")
		if dbPort == "" {
			dbPort = "3306"
		}
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local", dbUser, dbPass, dbHost, dbPort, dbName)
		log.Println("Connecting via TCP (Local)...")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("⚠️ DB Open Warning: %v", err)
		// ここではFatalにせず、後でエラー処理をする
	}
	defer db.Close()

	// ★修正点: ここでFatal(強制終了)させない！
	// サーバー起動前にPingでコケるとCloud Runが「起動失敗」とみなすため。
	if err := db.Ping(); err != nil {
		log.Printf("⚠️ WARNING: データベース接続に失敗しました: %v", err)
		log.Printf("⚠️ DSN(マスク済み): %s:****@...", dbUser)
	} else {
		log.Println("✅ Successfully connected to the database!")
	}

	// 3. ルーティング設定
	mux := http.NewServeMux()

	// ヘルスチェック & DB状態確認
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Cloud Run is running, but DB Connection Failed: %v", err)
			return
		}
		w.Write([]byte("Hello from Cloud Run! DB Connection is OK."))
	})

	// ① /api/items
	mux.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": 1, "name": "Test Item (Mock)", "price": 100},
		})
	})

	// ② /api/categories
	mux.HandleFunc("/api/categories", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{"Electronics", "Books", "Clothing"})
	})

	// ③ /api/auth/me
	mux.HandleFunc("/api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    1,
			"name":  "Test User",
			"email": "test@example.com",
		})
	})

	// 4. サーバー起動
	log.Printf("Server listening on port %s", port)
	// CORS設定でラップして起動
	if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// ==========================================
// 5. CORS設定
// ==========================================
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
