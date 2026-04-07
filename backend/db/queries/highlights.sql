-- name: CreateHighlight :one
INSERT INTO highlights (user_id, book_id, book_title, book_author, asin, content, location, highlighted_at, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetHighlightByID :one
SELECT * FROM highlights
WHERE id = $1 AND user_id = $2
LIMIT 1;

-- name: ListHighlightsByUserID :many
SELECT * FROM highlights
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountHighlightsByUserID :one
SELECT COUNT(*) FROM highlights
WHERE user_id = $1;

-- name: DeleteHighlight :exec
DELETE FROM highlights
WHERE id = $1 AND user_id = $2;
