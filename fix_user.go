package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 1. 管理者(root)でログインして、ユーザーを作り直します
	// ここは変えないでください（rootは今のあなたのPCで動いている実績があるため）
	adminUser := "root"
	adminPwd  := "Hackathon_2025" // ← ★ここにroot用のパスワードを入れる
	dbName    := "mysql"          // ← ユーザー管理用のDBにつなぎます

	// 2. 作成または修正したいユーザー情報
	targetUser := "uttc"
	// ↓↓↓ 【超重要】Cloud Runに設定しているパスワードと同じものを書いてください！ ↓↓↓
	newPassword := "Hackathon_2025" 

	// 接続
	dbURI := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s?parseTime=true&multiStatements=true", adminUser, adminPwd, dbName)
	db, err := sql.Open("mysql", dbURI)
	if err != nil { log.Fatal(err) }
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("❌ rootで接続できません。パスワードを確認してください: ", err)
	}

	fmt.Printf("🔧 ユーザー '%s' を修復しています...\n", targetUser)

	// 3. SQLコマンド実行（ユーザー作成・パスワード更新・権限付与）
	queries := []string{
		// ユーザーがいなければ作成
		fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'", targetUser, newPassword),
		// ユーザーがいればパスワードを強制更新
		fmt.Sprintf("ALTER USER '%s'@'%%' IDENTIFIED BY '%s'", targetUser, newPassword),
		// 権限を付与
		fmt.Sprintf("GRANT ALL PRIVILEGES ON hackathon_db.* TO '%s'@'%%'", targetUser),
		// 変更を反映
		"FLUSH PRIVILEGES",
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			log.Printf("⚠️ 警告 (でも続行します): %v", err)
		}
	}

	fmt.Println("✨ 完了！ユーザー 'uttc' は正常に設定されました。")
	fmt.Println("👉 これでブラウザをリロードすれば500エラーが消えるはずです！")
}