ALTER TABLE highlights
  ADD COLUMN IF NOT EXISTS book_key TEXT;

ALTER TABLE highlights
  ADD COLUMN IF NOT EXISTS book_order_index INT;

UPDATE highlights h
SET book_key = CASE
    WHEN NULLIF(trim(coalesce(h.book_key, '')), '') IS NOT NULL THEN trim(h.book_key)
    WHEN NULLIF(trim(coalesce(h.asin, '')), '') IS NOT NULL THEN trim(h.asin)
    ELSE concat('metadata:', trim(coalesce(h.book_title, '')), ':', trim(coalesce(h.book_author, '')))
END
WHERE NULLIF(trim(coalesce(h.book_key, '')), '') IS NULL;

WITH missing AS (
  SELECT
    id,
    user_id,
    book_key,
    ROW_NUMBER() OVER (
      PARTITION BY user_id, book_key
      ORDER BY COALESCE(highlighted_at, created_at), created_at, id
    ) AS rn
  FROM highlights
  WHERE book_order_index IS NULL
    AND NULLIF(trim(coalesce(book_key, '')), '') IS NOT NULL
),
offsets AS (
  SELECT
    user_id,
    book_key,
    COALESCE(MAX(book_order_index), 0) AS offset_value
  FROM highlights
  WHERE book_order_index IS NOT NULL
    AND NULLIF(trim(coalesce(book_key, '')), '') IS NOT NULL
  GROUP BY user_id, book_key
)
UPDATE highlights h
SET book_order_index = missing.rn + COALESCE(offsets.offset_value, 0)
FROM missing
LEFT JOIN offsets
  ON offsets.user_id = missing.user_id
 AND offsets.book_key = missing.book_key
WHERE h.id = missing.id;

CREATE INDEX IF NOT EXISTS idx_highlights_user_book_status
  ON highlights(user_id, book_key, status);

CREATE UNIQUE INDEX IF NOT EXISTS uq_highlights_book_order_index
  ON highlights(user_id, book_key, book_order_index)
  WHERE book_order_index IS NOT NULL;
