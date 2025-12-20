package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 1. 設定（あなたのパスワードに書き換えてください）
	dbUser := "root"
	dbPwd  := "Hackathon_2025" 
	dbName := "hackathon" 
	
	// 読み込むファイル名（SQL.text または data.sql など）
	fileName := "SQL.text"

	// 2. DB接続
	// 【重要】multiStatements=true をつけることで、大量のSQLをまとめて実行できます
	dbURI := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s?parseTime=true&multiStatements=true", dbUser, dbPwd, dbName)

	db, err := sql.Open("mysql", dbURI)
	if err != nil { log.Fatal(err) }
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("❌ プロキシが動いていないか、接続できません: ", err)
	}

	// 3. ファイルを読み込む
	fmt.Printf("📂 %s を読み込んでいます...\n", fileName)
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatal("❌ ファイルが見つかりません。フォルダに置いてありますか？: ", err)
	}

	sqlQueries := string(content)

	// 4. 実行！
	fmt.Println("🚀 データを流し込んでいます（数秒かかります）...")
	_, err = db.Exec(sqlQueries)
	if err != nil {
		// エラーが長すぎる場合があるので、最初の一部だけ表示
		errMsg := fmt.Sprintf("%v", err)
		if len(errMsg) > 200 { errMsg = errMsg[:200] + "..." }
		log.Fatalf("❌ SQL実行エラー: %s", errMsg)
	}

	fmt.Println("✨ 完了！すべてのデータが保存されました。")
}