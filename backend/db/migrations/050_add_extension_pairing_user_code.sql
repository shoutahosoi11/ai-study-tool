ALTER TABLE extension_pairings
  ADD COLUMN IF NOT EXISTS user_code TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS uq_extension_pairings_user_code_active
  ON extension_pairings(user_code)
  WHERE user_code IS NOT NULL AND used_at IS NULL;
