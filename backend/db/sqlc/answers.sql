-- name: UpsertAnswer :one
INSERT INTO answers (
    user_id,
    question_id,
    user_answer,
    is_correct,
    score,
    feedback,
    grader_model
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (user_id, question_id) DO UPDATE SET
    user_answer  = EXCLUDED.user_answer,
    is_correct   = EXCLUDED.is_correct,
    score        = EXCLUDED.score,
    feedback     = EXCLUDED.feedback,
    grader_model = EXCLUDED.grader_model,
    updated_at   = NOW()
RETURNING *;
