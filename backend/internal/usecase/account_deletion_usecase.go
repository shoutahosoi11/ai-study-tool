package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type AccountDeletionUsecase struct {
	userDeleter domain.UserAccountDeleter
	authManager domain.AuthAccountManager
}

func NewAccountDeletionUsecase(
	userDeleter domain.UserAccountDeleter,
	authManager domain.AuthAccountManager,
) *AccountDeletionUsecase {
	return &AccountDeletionUsecase{
		userDeleter: userDeleter,
		authManager: authManager,
	}
}

// DeleteAccount はユーザーの全データを削除する。
//
// 削除順序の意図:
//  1. DB削除を先に行う。admin_identities の ON DELETE RESTRICT により
//     管理者アカウントはここで拒否されるため、認証基盤側を先に消して
//     ログイン不能な管理者を作ってしまう事故を防ぐ。
//  2. DB削除成功後にFirebaseのセッション失効とユーザー削除を行う。
//     この段階の失敗は「PII削除は完了、認証レコードのみ残存」なので、
//     エラーにせずログに残して運用で回収する（DBユーザーが既にいない
//     ため、エラーを返してもユーザーは再試行できない）。
func (u *AccountDeletionUsecase) DeleteAccount(ctx context.Context, userID uuid.UUID, firebaseUID string) error {
	if userID == uuid.Nil || firebaseUID == "" {
		return domain.ErrInvalidInput
	}

	if err := u.userDeleter.DeleteByID(ctx, userID); err != nil {
		return fmt.Errorf("account deletion usecase: delete user data: %w", err)
	}

	if err := u.authManager.RevokeRefreshTokens(ctx, firebaseUID); err != nil {
		slog.Error("account_deletion_event=revoke_failed",
			"user_id", userID.String(), "firebase_uid", firebaseUID, "error", err)
	}
	if err := u.authManager.DeleteUser(ctx, firebaseUID); err != nil {
		slog.Error("account_deletion_event=auth_user_cleanup_failed",
			"user_id", userID.String(), "firebase_uid", firebaseUID, "error", err)
	}

	return nil
}
