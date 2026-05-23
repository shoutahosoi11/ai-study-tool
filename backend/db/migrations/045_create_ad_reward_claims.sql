CREATE TABLE IF NOT EXISTS ad_reward_claims (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  nonce TEXT NOT NULL,
  rewarded_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(provider, nonce)
);

CREATE INDEX IF NOT EXISTS idx_ad_reward_claims_user_created
  ON ad_reward_claims(user_id, created_at DESC);
