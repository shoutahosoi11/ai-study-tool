DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'chk_question_generation_jobs_status'
  ) THEN
    ALTER TABLE question_generation_jobs
      ADD CONSTRAINT chk_question_generation_jobs_status
      CHECK (status IN ('queued', 'processing', 'completed', 'failed', 'enqueue_failed')) NOT VALID;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'chk_question_generation_jobs_reason'
  ) THEN
    ALTER TABLE question_generation_jobs
      ADD CONSTRAINT chk_question_generation_jobs_reason
      CHECK (reason IN ('highlight_batch_threshold', 'all_unanswered_consumed', 'manual_selection')) NOT VALID;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'chk_question_generation_jobs_retry_count'
  ) THEN
    ALTER TABLE question_generation_jobs
      ADD CONSTRAINT chk_question_generation_jobs_retry_count
      CHECK (retry_count >= 0) NOT VALID;
  END IF;
END $$;

WITH ranked_active_jobs AS (
  SELECT
    id,
    ROW_NUMBER() OVER (
      PARTITION BY user_id, book_key
      ORDER BY
        CASE status
          WHEN 'processing' THEN 1
          WHEN 'queued' THEN 2
          WHEN 'enqueue_failed' THEN 3
          ELSE 4
        END,
        created_at DESC
    ) AS rank
  FROM question_generation_jobs
  WHERE status IN ('queued', 'processing', 'enqueue_failed')
)
UPDATE question_generation_jobs AS j
SET status = 'failed',
    failed_at = COALESCE(j.failed_at, NOW()),
    last_error = LEFT(trim(coalesce(j.last_error, '') || ' superseded duplicate active job before unique index'), 500),
    updated_at = NOW()
FROM ranked_active_jobs AS ranked
WHERE j.id = ranked.id
  AND ranked.rank > 1;

DROP INDEX IF EXISTS uq_question_generation_jobs_active;
CREATE UNIQUE INDEX IF NOT EXISTS uq_question_generation_jobs_active
  ON question_generation_jobs(user_id, book_key)
  WHERE status IN ('queued', 'processing', 'enqueue_failed');

CREATE INDEX IF NOT EXISTS idx_questions_user_active_highlight
  ON questions(user_id, highlight_id)
  WHERE highlight_id IS NOT NULL AND superseded_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_highlights_user_updated_at
  ON highlights(user_id, updated_at);
