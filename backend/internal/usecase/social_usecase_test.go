package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type fakeSocialRepository struct {
	domain.SocialRepository

	followed       bool
	createdComment *domain.Comment
	listInput      *domain.ListCommentsInput
}

func (f *fakeSocialRepository) Follow(ctx context.Context, followerID, followeeID string) error {
	f.followed = true
	return nil
}

func (f *fakeSocialRepository) Unfollow(ctx context.Context, followerID, followeeID string) error {
	return nil
}

func (f *fakeSocialRepository) CreateComment(ctx context.Context, comment *domain.Comment) (*domain.Comment, error) {
	f.createdComment = comment
	return comment, nil
}

func (f *fakeSocialRepository) ListComments(ctx context.Context, input domain.ListCommentsInput) ([]*domain.Comment, error) {
	f.listInput = &input
	return nil, nil
}

func requireSocialValidationError(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.HasPrefix(err.Error(), "validation:") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestFollowRejectsSelfFollow(t *testing.T) {
	repo := &fakeSocialRepository{}
	u := NewSocialUsecase(repo)

	err := u.Follow(context.Background(), "user-1", "user-1")

	requireSocialValidationError(t, err)
	if repo.followed {
		t.Fatal("repository Follow should not be called on self follow")
	}
}

func TestUnfollowRejectsSelfUnfollow(t *testing.T) {
	u := NewSocialUsecase(&fakeSocialRepository{})

	err := u.Unfollow(context.Background(), "user-1", "user-1")

	requireSocialValidationError(t, err)
}

func TestCreateCommentRejectsBlankContent(t *testing.T) {
	u := NewSocialUsecase(&fakeSocialRepository{})

	_, err := u.CreateComment(context.Background(), &domain.Comment{Content: "   "})

	requireSocialValidationError(t, err)
}

func TestCreateCommentRejectsLongContent(t *testing.T) {
	u := NewSocialUsecase(&fakeSocialRepository{})

	_, err := u.CreateComment(context.Background(), &domain.Comment{Content: strings.Repeat("あ", 501)})

	requireSocialValidationError(t, err)
}

func TestCreateCommentTrimsContent(t *testing.T) {
	repo := &fakeSocialRepository{}
	u := NewSocialUsecase(repo)

	if _, err := u.CreateComment(context.Background(), &domain.Comment{Content: "  コメント  "}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createdComment.Content != "コメント" {
		t.Fatalf("expected trimmed content, got %q", repo.createdComment.Content)
	}
}

func TestListCommentsClampsPagination(t *testing.T) {
	repo := &fakeSocialRepository{}
	u := NewSocialUsecase(repo)

	cases := []struct {
		limit, offset         int
		wantLimit, wantOffset int
	}{
		{limit: 0, offset: 0, wantLimit: 20, wantOffset: 0},
		{limit: -3, offset: -9, wantLimit: 20, wantOffset: 0},
		{limit: 100, offset: 5, wantLimit: 50, wantOffset: 5},
	}
	for _, tc := range cases {
		if _, err := u.ListComments(context.Background(), "post-1", tc.limit, tc.offset); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.listInput.Limit != tc.wantLimit || repo.listInput.Offset != tc.wantOffset {
			t.Fatalf("limit=%d offset=%d: expected (%d,%d), got (%d,%d)",
				tc.limit, tc.offset, tc.wantLimit, tc.wantOffset, repo.listInput.Limit, repo.listInput.Offset)
		}
	}
}
