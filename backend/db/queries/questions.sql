-- name: CreateQuestionGeneration :one
INSERT INTO question_generations (user_id, source_type, source_id, prompt_used, model_used)
VALUES (, , , , )
RETURNING *;

-- name: CreateQuestion :one
INSERT INTO questions (
    user_id, field_id, source_type, question_type,
    body, options, correct_answer, explanation,
    custom_instruction, is_ai_generated, generation_id
)
VALUES (, , , , , , , , , , )
RETURNING *;

-- name: GetQuestionByID :one
SELECT
    q.id, q.user_id, q.field_id, q.source_type, q.question_type,
    q.body, q.options, q.correct_answer, q.explanation,
    q.custom_instruction, q.answer_count, q.correct_count,
    q.is_ai_generated, q.generation_id, q.created_at, q.updated_at
FROM questions q
WHERE q.id = $1
LIMIT 1;

-- name: UpdateQuestionStats :exec
UPDATE questions
SET
    answer_count  = answer_count + 1,
    correct_count = correct_count + CASE WHEN $2::boolean THEN 1 ELSE 0 END,
    updated_at    = NOW()
WHERE id = $1;

-- name: ListUserQuestions :many
SELECT
    q.id, q.user_id, q.field_id, q.source_type, q.question_type,
    q.body, q.options, q.correct_answer, q.explanation,
    q.custom_instruction, q.answer_count, q.correct_count,
    q.is_ai_generated, q.generation_id, q.created_at, q.updated_at
FROM questions q
WHERE q.user_id = $1
ORDER BY q.created_at DESC
LIMIT $2 OFFSET $3;
