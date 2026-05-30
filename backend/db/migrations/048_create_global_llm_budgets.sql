CREATE TABLE IF NOT EXISTS global_llm_budgets (
  budget_date DATE PRIMARY KEY,
  max_requests INT NOT NULL,
  used_requests INT NOT NULL DEFAULT 0,
  max_estimated_cost_yen INT NOT NULL,
  used_estimated_cost_yen INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (max_requests >= 0),
  CHECK (used_requests >= 0),
  CHECK (max_estimated_cost_yen >= 0),
  CHECK (used_estimated_cost_yen >= 0)
);

CREATE TABLE IF NOT EXISTS llm_usage_logs (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  job_id UUID REFERENCES question_generation_jobs(id) ON DELETE SET NULL,
  provider TEXT NOT NULL,
  model TEXT NOT NULL,
  input_tokens INT NOT NULL DEFAULT 0,
  output_tokens INT NOT NULL DEFAULT 0,
  estimated_cost_yen NUMERIC NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_user_created
  ON llm_usage_logs(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_job_id
  ON llm_usage_logs(job_id);
