CREATE TABLE IF NOT EXISTS extension_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT UNIQUE NOT NULL,
  name TEXT,
  scopes TEXT[] NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_used_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_extension_tokens_user_created
  ON extension_tokens(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_extension_tokens_active_hash
  ON extension_tokens(token_hash)
  WHERE revoked_at IS NULL;
