package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// あなたの設定
	dbUser := "root"
	dbPwd  := "Hackathon_2025" 
	dbName := "hackathon" 

	// 接続
	dbURI := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s?parseTime=true", dbUser, dbPwd, dbName)
	db, err := sql.Open("mysql", dbURI)
	if err != nil { log.Fatal(err) }
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("❌ 接続できません。プロキシは動いていますか？: ", err)
	}

	fmt.Println("🧹 お掃除を開始します...")

	// 1. 外キー制約を一時的に無効化（これをしないと消せないデータがあるため）
	_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 0")

	// 2. 全部のテーブル名を取得
	rows, err := db.Query("SHOW TABLES")
	if err != nil { log.Fatal(err) }
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err == nil {
			tables = append(tables, table)
		}
	}

	// 3. 全テーブルを空にする (TRUNCATE)
	for _, table := range tables {
		fmt.Printf("🔥 テーブル '%s' を空にしています...\n", table)
		_, err := db.Exec("TRUNCATE TABLE " + table)
		if err != nil {
			log.Printf("⚠️ %s の削除に失敗: %v\n", table, err)
		}
	}

	// 4. 外キー制約を戻す
	_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	fmt.Println("✨ 完了！データベースは完全に空っぽ（新品）になりました。")
	fmt.Println("👉 次に import_db.go を実行して、データを流し込んでください。")
}