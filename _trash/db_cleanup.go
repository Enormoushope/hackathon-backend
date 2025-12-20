// db_cleanup.go
// 商品テーブルの異常データ（商品名が空・ダミー値、画像URLが同じ、価格が0円等）を一括削除するスクリプト
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main_cleanup() {
	dbUser := os.Getenv("MYSQL_USER")
	dbPass := os.Getenv("MYSQL_PASSWORD")
	dbName := os.Getenv("MYSQL_DATABASE")
	instanceConnectionName := os.Getenv("INSTANCE_CONNECTION_NAME")

	var dsn string
	if instanceConnectionName != "" {
		dsn = fmt.Sprintf("%s:%s@unix(/cloudsql/%s)/%s?parseTime=true&loc=Local", dbUser, dbPass, instanceConnectionName, dbName)
	} else {
		dbHost := os.Getenv("MYSQL_HOST")
		if dbHost == "" {
			dbHost = "127.0.0.1"
		}
		dbPort := os.Getenv("DB_PORT")
		if dbPort == "" {
			dbPort = "3306"
		}
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local", dbUser, dbPass, dbHost, dbPort, dbName)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 1. 商品名が空・ダミー値
	res1, err := db.Exec(`DELETE FROM items WHERE title IS NULL OR TRIM(title) = '' OR LOWER(title) LIKE '%dummy%' OR title LIKE '%テスト%'`)
	if err != nil {
		panic(err)
	}
	cnt1, _ := res1.RowsAffected()
	fmt.Printf("商品名異常データ削除: %d件\n", cnt1)

	// 2. 画像URLが空・ダミー値
	res2, err := db.Exec(`DELETE FROM items WHERE image_url IS NULL OR TRIM(image_url) = '' OR LOWER(image_url) LIKE '%noimage%' OR LOWER(image_url) LIKE '%dummy%'`)
	if err != nil {
		panic(err)
	}
	cnt2, _ := res2.RowsAffected()
	fmt.Printf("画像URL異常データ削除: %d件\n", cnt2)

	// 3. 価格が0円や異常値
	res3, err := db.Exec(`DELETE FROM items WHERE price < 300 OR price > 9999999`)
	if err != nil {
		panic(err)
	}
	cnt3, _ := res3.RowsAffected()
	fmt.Printf("価格異常データ削除: %d件\n", cnt3)

	// 4. 同一画像URLが5件以上ある場合は重複分を削除
	res4, err := db.Exec(`DELETE FROM items WHERE image_url IN (SELECT image_url FROM (SELECT image_url, COUNT(*) as cnt FROM items GROUP BY image_url HAVING cnt >= 5) t)`)
	if err != nil {
		panic(err)
	}
	cnt4, _ := res4.RowsAffected()
	fmt.Printf("画像重複データ削除: %d件\n", cnt4)

	fmt.Println("クレンジング完了")
}
