package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/repository"
)

type postRepository struct {
	db *sql.DB
}

func NewPostRepository(db *sql.DB) repository.PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) GetTimeline(ctx context.Context, params domain.TimelineParams) ([]*domain.TimelinePost, error) {
	query := `
WITH scored AS (
  SELECT
    p.id,
    MIN(
      CASE
        WHEN f.follower_id IS NOT NULL THEN 1
        WHEN ub.user_id    IS NOT NULL THEN 2
        WHEN ui.user_id    IS NOT NULL THEN 3
      END
    ) AS score
  FROM posts p
  LEFT JOIN follows        f  ON p.user_id  = f.followee_id  AND f.follower_id = $1
  LEFT JOIN user_books     ub ON p.book_id  = ub.book_id     AND ub.user_id    = $1
  LEFT JOIN user_interests ui ON p.field_id = ui.field_id    AND ui.user_id    = $1
  WHERE p.user_id != $1
    AND (f.follower_id IS NOT NULL OR ub.user_id IS NOT NULL OR ui.user_id IS NOT NULL)
  GROUP BY p.id
)
SELECT
  p.id, p.user_id, p.question_id, p.note_id, p.book_id, p.field_id,
  p.type, p.repost_count, p.like_count, p.comment_count,
  p.created_at, p.updated_at,
  s.score,
  u.username, u.display_name, u.avatar_url,
  b.title   AS book_title,
  fi.name   AS field_name
FROM scored s
JOIN  posts  p  ON s.id        = p.id
JOIN  users  u  ON p.user_id   = u.id
LEFT JOIN books  b  ON p.book_id   = b.id
LEFT JOIN fields fi ON p.field_id  = fi.id
ORDER BY s.score ASC, p.created_at DESC
LIMIT  $2
OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, params.UserID, params.Limit, params.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*domain.TimelinePost
	for rows.Next() {
		tp := &domain.TimelinePost{}
		err := rows.Scan(
			&tp.ID, &tp.UserID, &tp.QuestionID, &tp.NoteID, &tp.BookID, &tp.FieldID,
			&tp.Type, &tp.RepostCount, &tp.LikeCount, &tp.CommentCount,
			&tp.CreatedAt, &tp.UpdatedAt,
			&tp.Score,
			&tp.Username, &tp.DisplayName, &tp.AvatarURL,
			&tp.BookTitle, &tp.FieldName,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, tp)
	}
	return posts, rows.Err()
}

func (r *postRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.TimelinePost, error) {
	query := `
SELECT
  p.id, p.user_id, p.question_id, p.note_id, p.book_id, p.field_id,
  p.type, p.repost_count, p.like_count, p.comment_count,
  p.created_at, p.updated_at,
  0 AS score,
  u.username, u.display_name, u.avatar_url,
  b.title   AS book_title,
  fi.name   AS field_name
FROM posts p
JOIN  users  u  ON p.user_id  = u.id
LEFT JOIN books  b  ON p.book_id  = b.id
LEFT JOIN fields fi ON p.field_id = fi.id
WHERE p.id = $1
LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, id)
	tp := &domain.TimelinePost{}
	err := row.Scan(
		&tp.ID, &tp.UserID, &tp.QuestionID, &tp.NoteID, &tp.BookID, &tp.FieldID,
		&tp.Type, &tp.RepostCount, &tp.LikeCount, &tp.CommentCount,
		&tp.CreatedAt, &tp.UpdatedAt,
		&tp.Score,
		&tp.Username, &tp.DisplayName, &tp.AvatarURL,
		&tp.BookTitle, &tp.FieldName,
	)
	if err != nil {
		return nil, err
	}
	return tp, nil
}

func (r *postRepository) Create(ctx context.Context, input domain.CreatePostInput) (*domain.Post, error) {
	query := `
INSERT INTO posts (user_id, question_id, note_id, book_id, field_id, type)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, question_id, note_id, book_id, field_id, type, repost_count, like_count, comment_count, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query,
		input.UserID, input.QuestionID, input.NoteID, input.BookID, input.FieldID, input.Type,
	)
	p := &domain.Post{}
	err := row.Scan(
		&p.ID, &p.UserID, &p.QuestionID, &p.NoteID, &p.BookID, &p.FieldID,
		&p.Type, &p.RepostCount, &p.LikeCount, &p.CommentCount,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *postRepository) IncrementLike(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE posts SET like_count = like_count + 1 WHERE id = $1`, id)
	return err
}

func (r *postRepository) DecrementLike(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE posts SET like_count = GREATEST(like_count - 1, 0) WHERE id = $1`, id)
	return err
}

func (r *postRepository) IncrementRepost(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE posts SET repost_count = repost_count + 1 WHERE id = $1`, id)
	return err
}

func (r *postRepository) IncrementComment(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE posts SET comment_count = comment_count + 1 WHERE id = $1`, id)
	return err
}
