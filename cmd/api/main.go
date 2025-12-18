package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	httphandler "github.com/xyz77/hackathon/backend/internal/interface/http"
	_ "modernc.org/sqlite"
)

func main() {
	// Load .env file if exists
	_ = godotenv.Load()

	// Get configuration from environment variables
	// PORT は Cloud Run が 8080 で自動設定（空なら 8081）
	port := getEnv("PORT", "8080")
	projectID := getEnv("GOOGLE_CLOUD_PROJECT_ID", "citric-earth-477705-r6")
	location := getEnv("GOOGLE_CLOUD_LOCATION", "us-central1")

	// Initialize database (MySQL via Cloud Run env, fallback to sqlite for local dev)
	db, err := openDatabase()
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Initialize database schema and data
	// SKIP_SEED=true to skip seeding (useful for Cloud Run cold starts)
	skipSeed := os.Getenv("SKIP_SEED") == "true"
	if err := initializeDatabaseWithOptions(db, skipSeed); err != nil {
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
func openDatabase() (*sql.DB, error) {
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPwd := os.Getenv("MYSQL_PWD")
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlDB := os.Getenv("MYSQL_DATABASE")

	if mysqlUser != "" && mysqlHost != "" && mysqlDB != "" {
		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&loc=Local", mysqlUser, mysqlPwd, mysqlHost, mysqlDB)
		log.Printf("Connecting to MySQL at %s (db=%s)\n", mysqlHost, mysqlDB)
		return sql.Open("mysql", dsn)
	}

	// Fallback: sqlite (Cloud Run非rootで書き込み可能な /tmp をデフォルトにする)
	dbPath := getEnv("DATABASE_PATH", "/tmp/hackathon.db")
	log.Printf("MYSQL_* env not found, falling back to sqlite at %s\n", dbPath)
	return sql.Open("sqlite", dbPath)
}

func initializeDatabase(db *sql.DB) error {
	return initializeDatabaseWithOptions(db, false)
}

func initializeDatabaseWithOptions(db *sql.DB, skipSeed bool) error {
	// Create tables
	log.Println("Creating tables...")
	if err := createTables(db); err != nil {
		return err
	}
	log.Println("Tables created")

	// Backfill missing columns
	log.Println("Ensuring user columns...")
	if err := ensureUserColumns(db); err != nil {
		return err
	}
	log.Println("User columns ensured")

	log.Println("Ensuring transaction columns...")
	if err := ensureTransactionColumns(db); err != nil {
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
