CREATE TABLE IF NOT EXISTS stripe_events (
  event_id TEXT PRIMARY KEY,
  event_type TEXT NOT NULL,
  payload_hash TEXT NOT NULL,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS subscriptions (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  provider_user_id TEXT,
  subscription_id TEXT NOT NULL,
  product_id TEXT,
  status TEXT NOT NULL,
  current_period_end TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (provider IN ('stripe', 'apple', 'google')),
  CHECK (status IN ('active', 'trialing', 'past_due', 'canceled', 'expired', 'grace_period'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_subscriptions_user_provider_subscription
  ON subscriptions(user_id, provider, subscription_id);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_status
  ON subscriptions(user_id, status);

CREATE TABLE IF NOT EXISTS admob_ssv_events (
  transaction_id TEXT PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  ad_unit TEXT NOT NULL,
  reward_amount INT NOT NULL,
  reward_item TEXT NOT NULL,
  raw_query_hash TEXT NOT NULL,
  verified_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_admob_ssv_events_user_verified
  ON admob_ssv_events(user_id, verified_at DESC);
