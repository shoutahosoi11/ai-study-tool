DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'chk_highlight_import_queue_status'
  ) THEN
    ALTER TABLE highlight_import_queue
      ADD CONSTRAINT chk_highlight_import_queue_status
      CHECK (status IN ('queued', 'processing', 'completed', 'failed', 'enqueue_failed')) NOT VALID;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'chk_highlight_import_queue_retry_count'
  ) THEN
    ALTER TABLE highlight_import_queue
      ADD CONSTRAINT chk_highlight_import_queue_retry_count
      CHECK (retry_count >= 0) NOT VALID;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'chk_highlight_import_queue_payload_size'
  ) THEN
    ALTER TABLE highlight_import_queue
      ADD CONSTRAINT chk_highlight_import_queue_payload_size
      CHECK (octet_length(raw_payload::text) < 4194304) NOT VALID;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'chk_highlight_import_queue_source'
  ) THEN
    ALTER TABLE highlight_import_queue
      ADD CONSTRAINT chk_highlight_import_queue_source
      CHECK (source IN ('kindle')) NOT VALID;
  END IF;
END $$;

DROP INDEX IF EXISTS idx_highlight_import_queue_user_status;
CREATE INDEX IF NOT EXISTS idx_highlight_import_queue_user_status
  ON highlight_import_queue(user_id, status, created_at)
  WHERE status IN ('queued', 'processing', 'enqueue_failed');
