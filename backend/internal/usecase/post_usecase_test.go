package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type fakePostRepository struct {
	domain.PostRepository

	createInput    *domain.CreatePostInput
	timelineParams *domain.TimelineParams
	canView        bool
	canViewErr     error
}

func (f *fakePostRepository) Create(ctx context.Context, input domain.CreatePostInput) (*domain.Post, error) {
	f.createInput = &input
	return &domain.Post{}, nil
}

func (f *fakePostRepository) GetTimeline(ctx context.Context, params domain.TimelineParams) ([]*domain.TimelinePost, error) {
	f.timelineParams = &params
	return nil, nil
}

func (f *fakePostRepository) CanView(ctx context.Context, viewerID, postID uuid.UUID) (bool, error) {
	return f.canView, f.canViewErr
}

func requirePostValidationError(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.HasPrefix(err.Error(), "validation:") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestCreatePostRejectsQuestionPostWithoutQuestions(t *testing.T) {
	u := NewPostUsecase(&fakePostRepository{})

	_, err := u.CreatePost(context.Background(), domain.CreatePostInput{Type: "question", BookTitle: "本"})

	requirePostValidationError(t, err)
}

func TestCreatePostRejectsQuestionPostWithoutBookTitle(t *testing.T) {
	u := NewPostUsecase(&fakePostRepository{})

	_, err := u.CreatePost(context.Background(), domain.CreatePostInput{
		Type:      "question",
		Questions: []domain.PostQuestionItem{{}},
		BookTitle: "   ",
	})

	requirePostValidationError(t, err)
}

func TestCreatePostRejectsTooManyQuestions(t *testing.T) {
	u := NewPostUsecase(&fakePostRepository{})

	_, err := u.CreatePost(context.Background(), domain.CreatePostInput{
		Type:      "question",
		BookTitle: "本",
		Questions: make([]domain.PostQuestionItem, maxPostQuestions+1),
	})

	requirePostValidationError(t, err)
}

func TestCreatePostRejectsLongBody(t *testing.T) {
	u := NewPostUsecase(&fakePostRepository{})

	_, err := u.CreatePost(context.Background(), domain.CreatePostInput{
		Type: "text",
		Body: strings.Repeat("あ", 281),
	})

	requirePostValidationError(t, err)
}

func TestCreatePostRejectsLongQuestionNote(t *testing.T) {
	u := NewPostUsecase(&fakePostRepository{})

	_, err := u.CreatePost(context.Background(), domain.CreatePostInput{
		Type:      "question",
		BookTitle: "本",
		Questions: []domain.PostQuestionItem{{Note: strings.Repeat("あ", maxPostQuestionNoteLen+1)}},
	})

	requirePostValidationError(t, err)
}

func TestCreatePostDefaultsQuestionCountAndTrims(t *testing.T) {
	repo := &fakePostRepository{}
	u := NewPostUsecase(repo)

	_, err := u.CreatePost(context.Background(), domain.CreatePostInput{
		Type:      "question",
		BookTitle: "  本のタイトル  ",
		Body:      "  本文  ",
		Questions: []domain.PostQuestionItem{{Note: "  メモ  "}, {}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.createInput.QuestionCount != 2 {
		t.Fatalf("expected question count defaulted to 2, got %d", repo.createInput.QuestionCount)
	}
	if repo.createInput.BookTitle != "本のタイトル" || repo.createInput.Body != "本文" {
		t.Fatalf("expected trimmed fields, got %q %q", repo.createInput.BookTitle, repo.createInput.Body)
	}
	if repo.createInput.Questions[0].Note != "メモ" {
		t.Fatalf("expected trimmed note, got %q", repo.createInput.Questions[0].Note)
	}
}

func TestGetTimelineClampsLimit(t *testing.T) {
	repo := &fakePostRepository{}
	u := NewPostUsecase(repo)

	for _, badLimit := range []int{0, -5, 51} {
		if _, err := u.GetTimeline(context.Background(), uuid.New(), badLimit, 0); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.timelineParams.Limit != 20 {
			t.Fatalf("limit %d: expected clamp to 20, got %d", badLimit, repo.timelineParams.Limit)
		}
	}
}

func TestEnsureVisibleMapsInvisibleToNotFound(t *testing.T) {
	u := NewPostUsecase(&fakePostRepository{canView: false})

	err := u.EnsureVisible(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for invisible post, got %v", err)
	}
}

func TestEnsureVisiblePassesForVisiblePost(t *testing.T) {
	u := NewPostUsecase(&fakePostRepository{canView: true})

	if err := u.EnsureVisible(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("expected visible post to pass, got %v", err)
	}
}

func TestEnsureVisiblePropagatesRepositoryError(t *testing.T) {
	repoErr := errors.New("db down")
	u := NewPostUsecase(&fakePostRepository{canViewErr: repoErr})

	if err := u.EnsureVisible(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, repoErr) {
		t.Fatalf("expected repository error to propagate, got %v", err)
	}
}
