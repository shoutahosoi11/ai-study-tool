-- name: InsertFollow :execresult
INSERT INTO follows (follower_id, followee_id)
VALUES ($1, $2)
ON CONFLICT (follower_id, followee_id) DO NOTHING;

-- name: DeleteFollow :execresult
DELETE FROM follows
WHERE follower_id = $1 AND followee_id = $2;

-- name: InsertLike :execresult
INSERT INTO likes (user_id, post_id)
VALUES ($1, $2)
ON CONFLICT (user_id, post_id) DO NOTHING;

-- name: DeleteLike :execresult
DELETE FROM likes
WHERE user_id = $1 AND post_id = $2;

-- name: InsertRepost :execresult
INSERT INTO reposts (user_id, post_id)
VALUES ($1, $2)
ON CONFLICT (user_id, post_id) DO NOTHING;

-- name: DeleteRepost :execresult
DELETE FROM reposts
WHERE user_id = $1 AND post_id = $2;

-- name: IncrementLikeCount :exec
UPDATE posts SET like_count = GREATEST(like_count + $2, 0) WHERE id = $1;

-- name: IncrementRepostCount :exec
UPDATE posts SET repost_count = GREATEST(repost_count + $2, 0) WHERE id = $1;

-- name: IncrementCommentCount :exec
UPDATE posts SET comment_count = GREATEST(comment_count + $2, 0) WHERE id = $1;

-- name: InsertComment :one
INSERT INTO comments (post_id, user_id, content)
VALUES ($1, $2, $3)
RETURNING id, post_id, user_id, content, created_at;

-- name: ListCommentsByPostID :many
SELECT id, post_id, user_id, content, created_at
FROM comments
WHERE post_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
