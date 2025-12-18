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
   // ==========================================
   // 1. データベース接続設定 (ここが最重要)
   // ==========================================
   dbUser := os.Getenv("DB_USER")
   dbPass := os.Getenv("DB_PASSWORD")
   dbName := os.Getenv("DB_NAME")
   instanceConnectionName := os.Getenv("INSTANCE_CONNECTION_NAME") // Cloud Run用の変数

   var dsn string
   if instanceConnectionName != "" {
	   // 【Cloud Run用】Unixソケット接続
	   // フォーマット: user:password@unix(/cloudsql/接続名)/dbname
	   dsn = fmt.Sprintf("%s:%s@unix(/cloudsql/%s)/%s", dbUser, dbPass, instanceConnectionName, dbName)
	   log.Println("Connecting via Unix Socket...")
   } else {
	   // 【ローカル用】TCP接続
	   dbHost := os.Getenv("DB_HOST")
	   if dbHost == "" {
		   dbHost = "127.0.0.1"
	   }
	   dbPort := os.Getenv("DB_PORT")
	   if dbPort == "" {
		   dbPort = "3306"
	   }
	   dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPass, dbHost, dbPort, dbName)
	   log.Println("Connecting via TCP (Local)...")
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
	AllowedOrigins:   []string{"http://localhost:3000", "https://hackathon-frontend-jet.vercel.app"},
	   AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	   AllowedHeaders:   []string{"*"},
	   AllowCredentials: true,
   })
   handler := c.Handler(mux)

   // ==========================================
   // 4. サーバー起動 (ここも重要)
   // ==========================================
   port := os.Getenv("PORT")
   if port == "" {
	   port = "8080" // デフォルト
   }
   
   log.Printf("Server listening on port %s", port)
   // ":8080" とハードコードせず、変数 port を使うのが鉄則
   if err := http.ListenAndServe(":"+port, handler); err != nil {
	   log.Fatalf("Server failed to start: %v", err)
   }

	// Get configuration from environment variables
	// PORT は Cloud Run が 8080 で自動設定（空なら 8081）
	port := getEnv("PORT", "8080")
	projectID := getEnv("GOOGLE_CLOUD_PROJECT_ID", "citric-earth-477705-r6")
	location := getEnv("GOOGLE_CLOUD_LOCATION", "us-central1")

	   // Initialize database (MySQL via Cloud Run env, fallback to sqlite for local dev)
	   db, isMySQL, err := connectDB()
	   if err != nil {
		   log.Fatal("Failed to open database:", err)
	   }
	   defer db.Close()
// DB接続関数: 環境変数によってUnixソケットとTCPを切り替え
func connectDB() (*sql.DB, bool, error) {
   dbUser := os.Getenv("DB_USER")
   dbPass := os.Getenv("DB_PASSWORD")
   dbName := os.Getenv("DB_NAME")
   instanceConnectionName := os.Getenv("INSTANCE_CONNECTION_NAME")

   var dsn string
   if instanceConnectionName != "" {
	   // Cloud Run/本番用（Unix Socket）
	   dsn = fmt.Sprintf("%s:%s@unix(/cloudsql/%s)/%s?parseTime=true&loc=Local", dbUser, dbPass, instanceConnectionName, dbName)
	   log.Printf("Connecting to MySQL via Unix socket: /cloudsql/%s (db=%s)\n", instanceConnectionName, dbName)
	   return sql.Open("mysql", dsn), true, nil
   } else {
	   // ローカル用（TCP）
	   dbHost := os.Getenv("DB_HOST")
	   dbPort := os.Getenv("DB_PORT")
	   dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local", dbUser, dbPass, dbHost, dbPort, dbName)
	   log.Printf("Connecting to MySQL via TCP: %s:%s (db=%s)\n", dbHost, dbPort, dbName)
	   return sql.Open("mysql", dsn), true, nil
   }
}

	// Initialize database schema and data
	// SKIP_SEED=true to skip seeding (useful for Cloud Run cold starts)
	skipSeed := os.Getenv("SKIP_SEED") == "true"
	if err := initializeDatabaseWithOptions(db, skipSeed, isMySQL); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize Firebase Admin SDK
	ctx := context.Background()
	firebaseAuthManager := initializeFirebase(ctx)

	// Setup Gin router
	r := gin.Default()

	// CORS middleware (env-driven; defaults to allowing localhost and *.vercel.app)
	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
	if strings.TrimSpace(allowedOriginsEnv) != "" {
		// Comma-separated list of origins
		parts := strings.Split(allowedOriginsEnv, ",")
		origins := make([]string, 0, len(parts))
		for _, p := range parts {
			if v := strings.TrimSpace(p); v != "" {
				origins = append(origins, v)
			}
		}
		r.Use(cors.New(cors.Config{
			AllowOrigins:     origins,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
		}))
	} else {
		r.Use(cors.New(cors.Config{
			AllowOriginFunc: func(origin string) bool {
				if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "https://localhost:") {
					return true
				}
				if strings.HasSuffix(origin, ".vercel.app") || strings.HasSuffix(origin, ".vercel.dev") {
					return true
				}
				return false
			},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
		}))
	}

	// Initialize HTTP handler
	handler := httphandler.NewHTTPHandler(db, firebaseAuthManager)

	// Initialize VertexAI Manager
	vertexAIManager := httphandler.NewVertexAIManager(projectID, location)
	if err := vertexAIManager.Initialize(ctx); err != nil {
		log.Printf("Warning: Failed to initialize VertexAI: %v\n", err)
		log.Println("Please set GOOGLE_CLOUD_API_KEY environment variable")
	} else {
		log.Println("VertexAI initialized successfully")
		handler.SetVertexAIManager(vertexAIManager)
		defer vertexAIManager.Close()
	}

	// Register routes
	handler.RegisterRoutes(r)

	// Start server on 0.0.0.0 (Cloud Run requirement)
	addr := "0.0.0.0:" + port
	log.Printf("Server running on %s\n", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// openDatabase first tries MySQL using Cloud Run env vars, otherwise falls back to sqlite.
func openDatabase() (*sql.DB, bool, error) {
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPwd := os.Getenv("MYSQL_PWD")
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlDB := os.Getenv("MYSQL_DATABASE")

	if mysqlUser != "" && mysqlHost != "" && mysqlDB != "" {
		var dsn string
		if strings.HasPrefix(mysqlHost, "unix(") {
			dsn = fmt.Sprintf("%s:%s@%s/%s?parseTime=true&loc=Local", mysqlUser, mysqlPwd, mysqlHost, mysqlDB)
		} else {
			dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&loc=Local", mysqlUser, mysqlPwd, mysqlHost, mysqlDB)
		}
		log.Printf("Connecting to MySQL at %s (db=%s)\n", mysqlHost, mysqlDB)
		db, err := sql.Open("mysql", dsn)
		return db, true, err
	}

	// Fallback: sqlite (Cloud Run非rootで書き込み可能な /tmp をデフォルトにする)
	dbPath := getEnv("DATABASE_PATH", "/tmp/hackathon.db")
	log.Printf("MYSQL_* env not found, falling back to sqlite at %s\n", dbPath)
	db, err := sql.Open("sqlite", dbPath)
	return db, false, err
}

func initializeDatabase(db *sql.DB) error {
	return initializeDatabaseWithOptions(db, false, false)
}

func initializeDatabaseWithOptions(db *sql.DB, skipSeed, isMySQL bool) error {
	// Create tables
	log.Println("Creating tables...")
	if err := createTables(db, isMySQL); err != nil {
		return err
	}
	log.Println("Tables created")

	// Backfill missing columns
	log.Println("Ensuring user columns...")
	if err := ensureUserColumns(db, isMySQL); err != nil {
		return err
	}
	log.Println("User columns ensured")

	log.Println("Ensuring transaction columns...")
	if err := ensureTransactionColumns(db, isMySQL); err != nil {
		return err
	}
	log.Println("Transaction columns ensured")

	// Seed initial data (skip if flag is set)
	if skipSeed {
		log.Println("Skipping seed data (SKIP_SEED=true)")
		return nil
	}

	seedMode := getEnv("SEED_MODE", "full")
	switch seedMode {
	case "lite":
		log.Println("Seeding data (lite mode)...")
		if err := seedDataLite(db); err != nil {
			return err
		}
		log.Println("Seeding complete (lite)")
	default:
		log.Println("Seeding data (full mode)...")
		if err := seedData(db); err != nil {
			return err
		}
		log.Println("Seeding complete (full)")
	}

	// Sync counters
	log.Println("Syncing selling counts...")
	if err := syncSellingCounts(db); err != nil {
		return err
	}

	log.Println("Syncing transaction counts...")
	if err := syncTransactionCounts(db); err != nil {
		return err
	}

	log.Println("Syncing user ratings...")
	if err := syncUserRatings(db); err != nil {
		return err
	}

	log.Println("Syncing follower counts...")
	if err := syncFollowerCounts(db); err != nil {
		return err
	}
	log.Println("Database initialization complete")

	return nil
}

func initializeFirebase(ctx context.Context) *httphandler.FirebaseAuthManager {
	firebaseApp, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Println("Warning: Firebase Admin SDK not initialized. Admin features disabled.")
		log.Println("Error:", err)
		return nil
	}

	authClient, err := firebaseApp.Auth(ctx)
	if err != nil {
		log.Println("Warning: Failed to initialize Firebase Auth client:", err)
		return nil
	}

	return httphandler.NewFirebaseAuthManager(authClient)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
