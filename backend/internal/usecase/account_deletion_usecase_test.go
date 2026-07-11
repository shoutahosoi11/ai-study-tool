package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type fakeUserDeleter struct {
	deletedID uuid.UUID
	err       error
}

func (f *fakeUserDeleter) DeleteByID(ctx context.Context, id uuid.UUID) error {
	f.deletedID = id
	return f.err
}

type fakeAuthAccountManager struct {
	revokedUID string
	deletedUID string
	revokeErr  error
	deleteErr  error
}

func (f *fakeAuthAccountManager) RevokeRefreshTokens(ctx context.Context, uid string) error {
	f.revokedUID = uid
	return f.revokeErr
}

func (f *fakeAuthAccountManager) DeleteUser(ctx context.Context, uid string) error {
	f.deletedUID = uid
	return f.deleteErr
}

func TestDeleteAccountDeletesDBThenAuthUser(t *testing.T) {
	deleter := &fakeUserDeleter{}
	auth := &fakeAuthAccountManager{}
	u := NewAccountDeletionUsecase(deleter, auth)
	userID := uuid.New()

	if err := u.DeleteAccount(context.Background(), userID, "firebase-uid"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deleter.deletedID != userID {
		t.Fatalf("expected DB deletion of %s, got %s", userID, deleter.deletedID)
	}
	if auth.revokedUID != "firebase-uid" || auth.deletedUID != "firebase-uid" {
		t.Fatalf("expected auth revoke+delete for firebase-uid, got revoke=%q delete=%q",
			auth.revokedUID, auth.deletedUID)
	}
}

func TestDeleteAccountRejectsInvalidInput(t *testing.T) {
	u := NewAccountDeletionUsecase(&fakeUserDeleter{}, &fakeAuthAccountManager{})

	if err := u.DeleteAccount(context.Background(), uuid.Nil, "uid"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for nil user id, got %v", err)
	}
	if err := u.DeleteAccount(context.Background(), uuid.New(), ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty firebase uid, got %v", err)
	}
}

func TestDeleteAccountStopsWhenDBDeleteFails(t *testing.T) {
	auth := &fakeAuthAccountManager{}
	u := NewAccountDeletionUsecase(&fakeUserDeleter{err: domain.ErrAccountHasAdminRole}, auth)

	err := u.DeleteAccount(context.Background(), uuid.New(), "firebase-uid")

	if !errors.Is(err, domain.ErrAccountHasAdminRole) {
		t.Fatalf("expected admin-role error to propagate, got %v", err)
	}
	if auth.deletedUID != "" || auth.revokedUID != "" {
		t.Fatal("auth account must not be touched when DB deletion fails")
	}
}

func TestDeleteAccountSucceedsDespiteAuthCleanupFailure(t *testing.T) {
	auth := &fakeAuthAccountManager{deleteErr: errors.New("firebase down")}
	u := NewAccountDeletionUsecase(&fakeUserDeleter{}, auth)

	if err := u.DeleteAccount(context.Background(), uuid.New(), "firebase-uid"); err != nil {
		t.Fatalf("PII deletion succeeded; auth cleanup failure must not fail the request, got %v", err)
	}
}
