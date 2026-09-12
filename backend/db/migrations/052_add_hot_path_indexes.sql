-- Hot-path indexes identified in the performance review.
--
-- comments(post_id): GET /posts/:id/comments seq-scanned all comments.
-- questions(user_id, created_at): GET /questions sorted the whole table; the
--   existing partial indexes require highlight_id IS NOT NULL and do not apply.
-- question_generation_jobs(user_id, status): the pending-count and per-user
--   list queries bind status as a parameter array, so the partial unique index
--   cannot serve them, and the table grows forever (completed jobs are kept).
-- question_generation_jobs(created_at): admin job list ordering.
-- users stripe columns: webhook lookups by customer/subscription id scanned
--   the users table on every Stripe event.

CREATE INDEX IF NOT EXISTS idx_comments_post_id_created_at
    ON comments (post_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_questions_user_created
    ON questions (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_qgj_user_status
    ON question_generation_jobs (user_id, status);

CREATE INDEX IF NOT EXISTS idx_qgj_created_at
    ON question_generation_jobs (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_users_stripe_customer
    ON users (stripe_customer_id)
    WHERE stripe_customer_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_users_stripe_subscription
    ON users (stripe_subscription_id)
    WHERE stripe_subscription_id IS NOT NULL;
