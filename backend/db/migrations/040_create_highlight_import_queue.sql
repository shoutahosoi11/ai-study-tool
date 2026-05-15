CREATE TABLE highlight_import_queue (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  source TEXT NOT NULL CONSTRAINT chk_highlight_import_queue_source CHECK (source IN ('kindle')),
  raw_payload JSONB NOT NULL CONSTRAINT chk_highlight_import_queue_payload_size CHECK (octet_length(raw_payload::text) < 4194304),
  status TEXT NOT NULL DEFAULT 'queued' CONSTRAINT chk_highlight_import_queue_status CHECK (status IN ('queued', 'processing', 'completed', 'failed', 'enqueue_failed')),
  retry_count INT NOT NULL DEFAULT 0 CONSTRAINT chk_highlight_import_queue_retry_count CHECK (retry_count >= 0),
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  processing_started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  failed_at TIMESTAMPTZ
);

CREATE INDEX idx_highlight_import_queue_user_status
  ON highlight_import_queue(user_id, status, created_at)
  WHERE status IN ('queued', 'processing', 'enqueue_failed');
