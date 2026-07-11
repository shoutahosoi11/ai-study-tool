package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type mockQuestionRepository struct {
	save                         func(ctx context.Context, q *domain.Question, meta *domain.QuestionMeta) error
	listByCreatorID              func(ctx context.Context, creatorID string, limit int) ([]*domain.Question, error)
	listSavedByUserID            func(ctx context.Context, userID string, limit int) ([]*domain.SavedQuestion, error)
	listIncorrectByUserID        func(ctx context.Context, userID string, limit int) ([]*domain.IncorrectQuestion, error)
	listPreparedByHighlightIDs   func(ctx context.Context, userID string, highlightIDs []uuid.UUID, limit int) ([]*domain.Question, error)
	listPerspectivesByHighlight  func(ctx context.Context, userID string, highlightID uuid.UUID) ([]string, error)
	listUsedHighlightIDsByUserID func(ctx context.Context, userID string, highlightIDs []uuid.UUID) ([]uuid.UUID, error)
	findByID                     func(ctx context.Context, id string) (*domain.Question, *domain.QuestionMeta, *domain.QuestionStats, error)
	updateStats                  func(ctx context.Context, questionID string, isCorrect bool) error
	saveGeneration               func(ctx context.Context, userID, sourceType, sourceID, promptUsed, modelUsed string) (string, error)
	saveForUser                  func(ctx context.Context, userID, questionID, note string) error
}

func (m *mockQuestionRepository) Save(ctx context.Context, q *domain.Question, meta *domain.QuestionMeta) error {
	return m.save(ctx, q, meta)
}

func (m *mockQuestionRepository) SupersedeActiveQuestionsForHighlight(ctx context.Context, userID uuid.UUID, highlightID uuid.UUID) error {
	return nil
}

func (m *mockQuestionRepository) ListByCreatorID(ctx context.Context, creatorID string, limit int) ([]*domain.Question, error) {
	if m.listByCreatorID == nil {
		return make([]*domain.Question, 0), nil
	}
	return m.listByCreatorID(ctx, creatorID, limit)
}

func (m *mockQuestionRepository) ListSavedByUserID(ctx context.Context, userID string, limit int) ([]*domain.SavedQuestion, error) {
	if m.listSavedByUserID == nil {
		return make([]*domain.SavedQuestion, 0), nil
	}
	return m.listSavedByUserID(ctx, userID, limit)
}

func (m *mockQuestionRepository) ListIncorrectByUserID(ctx context.Context, userID string, limit int) ([]*domain.IncorrectQuestion, error) {
	if m.listIncorrectByUserID == nil {
		return make([]*domain.IncorrectQuestion, 0), nil
	}
	return m.listIncorrectByUserID(ctx, userID, limit)
}

func (m *mockQuestionRepository) ListPreparedByUserIDAndHighlightIDs(ctx context.Context, userID string, highlightIDs []uuid.UUID, limit int) ([]*domain.Question, error) {
	if m.listPreparedByHighlightIDs == nil {
		return make([]*domain.Question, 0), nil
	}
	return m.listPreparedByHighlightIDs(ctx, userID, highlightIDs, limit)
}

func (m *mockQuestionRepository) ListPerspectivesByHighlightID(ctx context.Context, userID string, highlightID uuid.UUID) ([]string, error) {
	if m.listPerspectivesByHighlight == nil {
		return make([]string, 0), nil
	}
	return m.listPerspectivesByHighlight(ctx, userID, highlightID)
}

func (m *mockQuestionRepository) ListUsedHighlightIDsByUserID(ctx context.Context, userID string, highlightIDs []uuid.UUID) ([]uuid.UUID, error) {
	if m.listUsedHighlightIDsByUserID == nil {
		return make([]uuid.UUID, 0), nil
	}
	return m.listUsedHighlightIDsByUserID(ctx, userID, highlightIDs)
}

func (m *mockQuestionRepository) FindByID(ctx context.Context, id string) (*domain.Question, *domain.QuestionMeta, *domain.QuestionStats, error) {
	return m.findByID(ctx, id)
}

func (m *mockQuestionRepository) GetByID(ctx context.Context, id string) (*domain.Question, error) {
	q, _, _, err := m.findByID(ctx, id)
	return q, err
}

func (m *mockQuestionRepository) UpdateStats(ctx context.Context, questionID string, isCorrect bool) error {
	return m.updateStats(ctx, questionID, isCorrect)
}

func (m *mockQuestionRepository) SaveGeneration(ctx context.Context, userID, sourceType, sourceID, promptUsed, modelUsed string) (string, error) {
	return m.saveGeneration(ctx, userID, sourceType, sourceID, promptUsed, modelUsed)
}

func (m *mockQuestionRepository) SaveForUser(ctx context.Context, userID, questionID, note string) error {
	if m.saveForUser == nil {
		return nil
	}
	return m.saveForUser(ctx, userID, questionID, note)
}

type mockQuestionSourceResolver struct {
	resolveHighlights func(ctx context.Context, userID string, sourceType domain.SourceType, sourceID string, bookTitle string, bookAuthor string) ([]*domain.Highlight, error)
}

func (m *mockQuestionSourceResolver) ResolveHighlights(ctx context.Context, userID string, sourceType domain.SourceType, sourceID string, bookTitle string, bookAuthor string) ([]*domain.Highlight, error) {
	if m.resolveHighlights == nil {
		return make([]*domain.Highlight, 0), nil
	}
	return m.resolveHighlights(ctx, userID, sourceType, sourceID, bookTitle, bookAuthor)
}

func TestSaveQuestionPassesAuthenticatedUserToRepository(t *testing.T) {
	repo := &mockQuestionRepository{
		findByID: func(ctx context.Context, id string) (*domain.Question, *domain.QuestionMeta, *domain.QuestionStats, error) {
			if id != "question-1" {
				t.Fatalf("unexpected question id: %s", id)
			}
			return &domain.Question{ID: id}, &domain.QuestionMeta{CreatorID: "user-1"}, nil, nil
		},
		saveForUser: func(ctx context.Context, userID, questionID, note string) error {
			if userID != "user-1" {
				t.Fatalf("unexpected user id: %s", userID)
			}
			if questionID != "question-1" {
				t.Fatalf("unexpected question id: %s", questionID)
			}
			if note != "my note" {
				t.Fatalf("unexpected note: %q", note)
			}
			return nil
		},
	}
	uc := usecase.NewQuestionUsecase(repo, &mockQuestionSourceResolver{})

	if err := uc.SaveQuestion(context.Background(), "user-1", "question-1", " my note "); err != nil {
		t.Fatalf("SaveQuestion failed: %v", err)
	}
}

func TestSaveQuestionRejectsNoteOnOtherUsersQuestion(t *testing.T) {
	called := false
	repo := &mockQuestionRepository{
		findByID: func(ctx context.Context, id string) (*domain.Question, *domain.QuestionMeta, *domain.QuestionStats, error) {
			return &domain.Question{ID: id}, &domain.QuestionMeta{CreatorID: "owner-user"}, nil, nil
		},
		saveForUser: func(ctx context.Context, userID, questionID, note string) error {
			called = true
			return nil
		},
	}
	uc := usecase.NewQuestionUsecase(repo, &mockQuestionSourceResolver{})

	err := uc.SaveQuestion(context.Background(), "other-user", "question-1", "not allowed")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if called {
		t.Fatal("SaveForUser must not be called for another user's noted question")
	}
}

func TestListSavedAndIncorrectQuestionsUseAuthenticatedUser(t *testing.T) {
	var savedUserID string
	var incorrectUserID string
	repo := &mockQuestionRepository{
		listSavedByUserID: func(ctx context.Context, userID string, limit int) ([]*domain.SavedQuestion, error) {
			savedUserID = userID
			return nil, nil
		},
		listIncorrectByUserID: func(ctx context.Context, userID string, limit int) ([]*domain.IncorrectQuestion, error) {
			incorrectUserID = userID
			return nil, nil
		},
	}
	uc := usecase.NewQuestionUsecase(repo, &mockQuestionSourceResolver{})

	if _, err := uc.ListSavedQuestions(context.Background(), "current-user", 20); err != nil {
		t.Fatalf("ListSavedQuestions failed: %v", err)
	}
	if _, err := uc.ListIncorrectQuestions(context.Background(), "current-user", 20); err != nil {
		t.Fatalf("ListIncorrectQuestions failed: %v", err)
	}
	if savedUserID != "current-user" || incorrectUserID != "current-user" {
		t.Fatalf("expected current user for saved/incorrect lists, got saved=%q incorrect=%q", savedUserID, incorrectUserID)
	}
}

func TestListPreparedQuestions_ReturnsPreparingWhenHighlightsPending(t *testing.T) {
	ctx := context.Background()
	highlightID := uuid.New()

	repo := &mockQuestionRepository{
		listPreparedByHighlightIDs: func(ctx context.Context, userID string, highlightIDs []uuid.UUID, limit int) ([]*domain.Question, error) {
			return make([]*domain.Question, 0), nil
		},
	}
	resolver := &mockQuestionSourceResolver{
		resolveHighlights: func(ctx context.Context, userID string, sourceType domain.SourceType, sourceID string, bookTitle string, bookAuthor string) ([]*domain.Highlight, error) {
			return []*domain.Highlight{
				{
					ID:      highlightID,
					Content: "共有ハイライト",
					Status:  domain.HighlightStatusPending,
				},
			}, nil
		},
	}

	uc := usecase.NewQuestionUsecase(repo, resolver)

	_, err := uc.ListPreparedQuestions(ctx, domain.GenerateQuestionsInput{
		CreatorID:     "user-123",
		SourceType:    domain.SourceTypeKindleBook,
		SourceID:      "metadata:book",
		QuestionCount: 3,
	})
	if !errors.Is(err, domain.ErrQuestionsPreparing) {
		t.Fatalf("expected ErrQuestionsPreparing, got %v", err)
	}
}
