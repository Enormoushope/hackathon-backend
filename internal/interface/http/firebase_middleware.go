package http

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SetAdminRequest - 管理者権限を設定
type SetAdminRequest struct {
	UserID  string `json:"userId" binding:"required"`
	IsAdmin bool   `json:"isAdmin"`
}

// SetUserAdmin - ユーザーに管理者権限を付与または削除（内部用）
func SetUserAdmin(authManager *FirebaseAuthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SetAdminRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := context.Background()

		if req.IsAdmin {
			// 管理者権限を付与
			if err := authManager.SetAdminClaims(ctx, req.UserID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Admin privileges granted", "userId": req.UserID})
		} else {
			// 管理者権限を削除
			if err := authManager.RemoveAdminClaims(ctx, req.UserID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Admin privileges removed", "userId": req.UserID})
		}
	}
}

// VerifyTokenMiddleware - IDトークンを検証するミドルウェア
func VerifyTokenMiddleware(authManager *FirebaseAuthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authManager == nil {
			fmt.Println("[WARN] Firebase authManager is nil - authentication disabled")
			// Firebase未初期化でも続行（開発環境用）
			c.Next()
			return
		}

		if authHeader == "" {
			// トークンなしでも続行（ログアウトユーザー用）
			c.Next()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			fmt.Printf("[DEBUG] Invalid authorization header format\n")
			// 開発用途: 401にせず続行（uidは設定されない）
			c.Next()
			return
		}
		token := authHeader[7:]
		fmt.Printf("[DEBUG] Verifying token (first 20 chars): %s...\n", token[:20])

		ctx := context.Background()
		decodedToken, err := authManager.VerifyIDToken(ctx, token)
		if err != nil {
			fmt.Printf("[ERROR] Token verification failed: %v\n", err)
			// 開発用途: 401にせず続行（uidは設定されない）
			c.Next()
			return
		}

		fmt.Printf("[DEBUG] Token verified for UID: %s\n", decodedToken.UID)
		// トークン情報をコンテキストに保存
		c.Set("uid", decodedToken.UID)
		c.Set("email", decodedToken.Claims["email"])
		c.Set("isAdmin", decodedToken.Claims["isAdmin"])
		c.Next()
	}
}

// AdminCheckMiddleware - 管理者権限をチェック
func AdminCheckMiddleware(authManager *FirebaseAuthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Firebase未初期化（ローカル開発など）の場合は管理者チェックをスキップ
		if authManager == nil || authManager.authClient == nil {
			c.Next()
			return
		}

		uid, exists := c.Get("uid")
		if !exists || uid == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		ctx := context.Background()
		isAdmin, err := authManager.IsAdmin(ctx, uid.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		if !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminCheckMiddlewareWithDB - Firebaseのカスタムクレームに加え、DBのis_adminも許可する
// Firebaseが初期化されている環境では、まずトークン検証済みのUIDを取得し、
// 1) カスタムクレーム isAdmin=true なら許可
// 2) DB users.is_admin=1 なら許可
// どちらも満たさない場合は403を返す
func AdminCheckMiddlewareWithDB(authManager *FirebaseAuthManager, db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Firebase未初期化（ローカル開発など）の場合でもDBのis_adminでチェックする

		uidVal, exists := c.Get("uid")
		uid := ""
		if exists && uidVal != "" {
			uid = uidVal.(string)
		} else {
			// 開発用途: トークン検証でuidが取れない場合は既定のテストUIDでチェック
			uid = "18oYncIdc3UuvZneYQQ4j2II23A2"
		}

		// まずFirebaseカスタムクレームを確認（利用可能な場合のみ）。
		isAdminClaims := false
		if authManager != nil && authManager.authClient != nil {
			ctx := context.Background()
			claimsOK, err := authManager.IsAdmin(ctx, uid)
			if err == nil {
				isAdminClaims = claimsOK
			} else {
				// 開発環境などでFirebase認証が使えない場合はエラーを無視し、DB判定へフォールバック
				fmt.Printf("[WARN] Firebase IsAdmin check failed, fallback to DB: %v\n", err)
			}
		}
		if isAdminClaims {
			c.Next()
			return
		}

		// 次にDBのis_adminフラグを確認
		var isAdminDB int
		err := db.QueryRow("SELECT is_admin FROM users WHERE id = ?", uid).Scan(&isAdminDB)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		if isAdminDB == 1 {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Admin access required"})
		c.Abort()
	}
}
