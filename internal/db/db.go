package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func InitDB() {
	user := os.Getenv("MYSQL_USER")
	pass := os.Getenv("MYSQL_PASSWORD")
	name := os.Getenv("MYSQL_DATABASE")
	host := os.Getenv("MYSQL_HOST")
	port := os.Getenv("DB_PORT")
	instanceConnectionName := os.Getenv("INSTANCE_CONNECTION_NAME")

	var dsn string

	// Cloud Run等で Cloud SQL Auth Proxy 経由（Unixソケット）で繋ぐ場合
	if instanceConnectionName != "" {
		// DSN形式: user:pass@unix(/cloudsql/INSTANCE_CONNECTION_NAME)/dbname
		dsn = fmt.Sprintf("%s:%s@unix(/cloudsql/%s)/%s?parseTime=true", 
			user, pass, instanceConnectionName, name)
		log.Printf("Connecting to Cloud SQL via Unix Socket...")
	} else {
		// ローカル開発用（TCP接続）
		if port == "" { port = "3306" }
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", 
			user, pass, host, port, name)
		log.Printf("Connecting to DB via TCP (%s)...", host)
	}

	var err error
	for i := 0; i < 10; i++ {
		DB, err = sql.Open("mysql", dsn)
		if err == nil {
			err = DB.Ping()
			if err == nil {
				log.Println("Database Connected!")
				return
			}
		}
		log.Printf("Retry %d/10: %v", i+1, err)
		time.Sleep(3 * time.Second)
	}
	log.Fatal("DB Connection Failed after 10 attempts")
}