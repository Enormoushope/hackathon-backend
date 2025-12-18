package http

import (
	"context"

	"firebase.google.com/go/v4/auth"
)

// FirebaseAuthManager はFirebase認証を管理
type FirebaseAuthManager struct {
	authClient *auth.Client
}

// NewFirebaseAuthManager はFirebaseAuthManagerを作成
func NewFirebaseAuthManager(authClient *auth.Client) *FirebaseAuthManager {
	return &FirebaseAuthManager{authClient: authClient}
}

// SetAdminClaims はユーザーのカスタムクレームにisAdmin=trueを設定
func (f *FirebaseAuthManager) SetAdminClaims(ctx context.Context, uid string) error {
	claims := map[string]interface{}{
		"isAdmin": true,
	}
	return f.authClient.SetCustomUserClaims(ctx, uid, claims)
}

// RemoveAdminClaims はユーザーのカスタムクレームからisAdminを削除
func (f *FirebaseAuthManager) RemoveAdminClaims(ctx context.Context, uid string) error {
	return f.authClient.SetCustomUserClaims(ctx, uid, nil)
}

// GetUserClaims はユーザーのカスタムクレームを取得
func (f *FirebaseAuthManager) GetUserClaims(ctx context.Context, uid string) (map[string]interface{}, error) {
	user, err := f.authClient.GetUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	return user.CustomClaims, nil
}

// VerifyIDToken はIDトークンを検証
func (f *FirebaseAuthManager) VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	return f.authClient.VerifyIDToken(ctx, idToken)
}

// IsAdmin はユーザーが管理者かどうかを確認
func (f *FirebaseAuthManager) IsAdmin(ctx context.Context, uid string) (bool, error) {
	claims, err := f.GetUserClaims(ctx, uid)
	if err != nil {
		return false, err
	}
	if claims == nil {
		return false, nil
	}
	isAdmin, ok := claims["isAdmin"].(bool)
	return ok && isAdmin, nil
}
