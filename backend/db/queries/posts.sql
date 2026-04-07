-- name: GetTimeline :many
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
  p.id,
  p.user_id,
  p.question_id,
  p.note_id,
  p.book_id,
  p.field_id,
  p.type,
  p.repost_count,
  p.like_count,
  p.comment_count,
  p.created_at,
  p.updated_at,
  s.score,
  u.username,
  u.display_name,
  u.avatar_url,
  b.title   AS book_title,
  fi.name   AS field_name
FROM scored s
JOIN  posts  p  ON s.id        = p.id
JOIN  users  u  ON p.user_id   = u.id
LEFT JOIN books  b  ON p.book_id   = b.id
LEFT JOIN fields fi ON p.field_id  = fi.id
ORDER BY s.score ASC, p.created_at DESC
LIMIT  $2
OFFSET $3;

-- name: CreatePost :one
INSERT INTO posts (user_id, question_id, note_id, book_id, field_id, type)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetPostByID :one
SELECT
  p.*,
  u.username,
  u.display_name,
  u.avatar_url,
  b.title   AS book_title,
  fi.name   AS field_name
FROM posts p
JOIN  users  u  ON p.user_id  = u.id
LEFT JOIN books  b  ON p.book_id  = b.id
LEFT JOIN fields fi ON p.field_id = fi.id
WHERE p.id = $1
LIMIT 1;

-- name: IncrementLikeCount :exec
UPDATE posts SET like_count = like_count + 1 WHERE id = $1;

-- name: DecrementLikeCount :exec
UPDATE posts SET like_count = GREATEST(like_count - 1, 0) WHERE id = $1;

-- name: IncrementRepostCount :exec
UPDATE posts SET repost_count = repost_count + 1 WHERE id = $1;

-- name: IncrementCommentCount :exec
UPDATE posts SET comment_count = comment_count + 1 WHERE id = $1;
