-- name: ListHighlightBooksByUserID :many
SELECT
    asin,
    COALESCE(MAX(book_title), '')::text AS book_title,
    COALESCE(MAX(book_author), '')::text AS book_author,
    COUNT(*) AS highlight_count,
    source
FROM highlights
WHERE user_id = $1
  AND asin IS NOT NULL
  AND source = $2
GROUP BY asin, source
ORDER BY MAX(highlighted_at) DESC NULLS LAST, MAX(created_at) DESC;

-- name: UpdateHighlightExplanation :one
UPDATE highlights
SET explanation = $3, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;
