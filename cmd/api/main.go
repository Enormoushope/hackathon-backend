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

type Item struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Price        int     `json:"price"`
	ImageURL     string  `json:"imageUrl"`
	IsSoldOut    bool    `json:"isSoldOut"`
	SellerID     *string `json:"sellerId"`
	LikeCount    int     `json:"likeCount"`
	ViewCount    int     `json:"viewCount"`
	CommentCount int     `json:"commentCount"`
	Category     *string `json:"category"`
	Description  *string `json:"description,omitempty"`
	Condition    *string `json:"condition,omitempty"`
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

	// ① /api/items
	mux.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// DBのカラム名に厳密に合わせて取得
		rows, err := db.Query("SELECT id, name, price, image_url, is_sold, user_id FROM items")
		if err != nil {
			log.Printf("Query Error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var items []Item
		for rows.Next() {
			var item Item
			if err := rows.Scan(
				&item.ID,        // id
				&item.Title,     // name → Title
				&item.Price,     // price
				&item.ImageURL,  // image_url
				&item.IsSoldOut, // is_sold → IsSoldOut
				&item.SellerID,  // user_id → SellerID
			); err != nil {
				log.Printf("Scan Error: %v", err)
				continue
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
		// 認証連携: Authorizationヘッダーからuidを抽出（例: Bearer <token>）
		uid := ""
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			// ここでJWT等からuidを抽出する処理を追加（例: "Bearer <uid>" の場合）
			// 本番ではJWT検証やFirebase Admin SDK等を使う
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				uid = authHeader[7:]
			}
		}
		if uid == "" {
			uid = r.Header.Get("X-User-Id")
		}
		if uid == "" {
			uid = r.URL.Query().Get("uid")
		}
		if uid == "" {
			uid = "1" // fallback: テスト用
		}
		var id, email string
		var name string
		err := db.QueryRow("SELECT id, username, email FROM users WHERE id = ?", uid).Scan(&id, &name, &email)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    id,
			"name":  name,
			"email": email,
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
