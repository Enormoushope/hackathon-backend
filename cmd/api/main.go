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

// Item構造体（DBのitemsテーブル用）
type Item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

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

	// 4. ルーティング設定
	mux := http.NewServeMux()

	// ヘルスチェック
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if err := db.Ping(); err != nil {
			fmt.Fprintf(w, "Cloud Run is running, but DB Connection Failed: %v", err)
			return
		}
		w.Write([]byte("Hello from Cloud Run! DB Connection is OK."))
	})

	// ① /api/items
	mux.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// DBからitemsテーブルのデータを取得
		rows, err := db.Query("SELECT id, name, price FROM items")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var items []Item
		for rows.Next() {
			var item Item
			if err := rows.Scan(&item.ID, &item.Name, &item.Price); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			items = append(items, item)
		}
		json.NewEncoder(w).Encode(items)
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

	// 5. サーバー起動
	log.Printf("Server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// CORSミドルウェア
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
