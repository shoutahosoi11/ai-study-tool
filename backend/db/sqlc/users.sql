-- name: GetUserByFirebaseUID :one
SELECT *
FROM users
WHERE firebase_uid = $1
LIMIT 1;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByUsername :one
SELECT *
FROM users
WHERE username = $1
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (
    firebase_uid,
    username,
    display_name,
    avatar_url,
    bio,
    university,
    faculty,
    grade,
    country,
    plan
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, 'free'
)
RETURNING *;

-- name: UpdateUser :one
UPDATE users SET
    username = CASE WHEN @set_username::boolean THEN @username::text ELSE username END,
    display_name = CASE WHEN @set_display_name::boolean THEN @display_name::text ELSE display_name END,
    avatar_url = CASE WHEN @set_avatar_url::boolean THEN sqlc.narg('avatar_url') ELSE avatar_url END,
    bio = CASE WHEN @set_bio::boolean THEN sqlc.narg('bio') ELSE bio END,
    university = CASE WHEN @set_university::boolean THEN sqlc.narg('university') ELSE university END,
    faculty = CASE WHEN @set_faculty::boolean THEN sqlc.narg('faculty') ELSE faculty END,
    grade = CASE WHEN @set_grade::boolean THEN sqlc.narg('grade') ELSE grade END,
    country = CASE WHEN @set_country::boolean THEN sqlc.narg('country') ELSE country END,
    updated_at = NOW()
WHERE id = @id
RETURNING *;

-- name: UpdateUserQuestionSettings :one
UPDATE users SET
    default_question_count = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
