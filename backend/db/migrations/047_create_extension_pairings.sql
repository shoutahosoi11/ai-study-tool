CREATE TABLE IF NOT EXISTS extension_pairings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  token_id UUID REFERENCES extension_tokens(id) ON DELETE SET NULL,
  scopes TEXT[] NOT NULL DEFAULT ARRAY['highlight:write', 'highlight:check', 'extension:revoke-self']::TEXT[],
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  approved_at TIMESTAMPTZ,
  used_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_extension_pairings_user_id ON extension_pairings(user_id);
CREATE INDEX IF NOT EXISTS idx_extension_pairings_expires_at ON extension_pairings(expires_at);
CREATE INDEX IF NOT EXISTS idx_extension_pairings_approved_unused
  ON extension_pairings(id)
  WHERE approved_at IS NOT NULL AND used_at IS NULL;
