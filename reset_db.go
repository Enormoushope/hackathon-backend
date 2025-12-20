package main

import (
	"database/sql"
	"fmt"
	"log"
	
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	// 設定読み込み
	_ = godotenv.Load()

	dbUser := "root"
	dbPwd := "Hackathon_2025"
	dbName := "hackathon"

	// プロキシ経由で接続
	dbURI := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s?parseTime=true", dbUser, dbPwd, dbName)

	db, err := sql.Open("mysql", dbURI)
	if err != nil { log.Fatal(err) }
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("❌ プロキシが動いていないか、接続できません: ", err)
	}

	// 消したいテーブル名
	tableName := "items" 

	fmt.Println("🔥 データを削除しています...")
	_, err = db.Exec("DELETE FROM " + tableName)
	if err != nil {
		log.Printf("❌ エラー: %v", err)
		fmt.Println("ヒント: テーブル名が 'items' じゃないかもしれません。")
	} else {
		fmt.Println("✨ 完了！データは空になりました。")
	}
}