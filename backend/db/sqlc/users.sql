-- name: GetUserByFirebaseUID :one
SELECT id, firebase_uid, username, display_name, avatar_url, bio, university, faculty, grade, country, plan, created_at, updated_at, default_question_count
FROM users
WHERE firebase_uid = $1
LIMIT 1;

-- name: GetUserByID :one
SELECT id, firebase_uid, username, display_name, avatar_url, bio, university, faculty, grade, country, plan, created_at, updated_at, default_question_count
FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByUsername :one
SELECT id, firebase_uid, username, display_name, avatar_url, bio, university, faculty, grade, country, plan, created_at, updated_at, default_question_count
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
RETURNING id, firebase_uid, username, display_name, avatar_url, bio, university, faculty, grade, country, plan, created_at, updated_at, default_question_count;

-- name: UpdateUser :one
UPDATE users SET
    username = $2,
    display_name = $3,
    avatar_url = $4,
    bio = $5,
    university = $6,
    faculty = $7,
    grade = $8,
    country = $9,
    updated_at = NOW()
WHERE id = $1
RETURNING id, firebase_uid, username, display_name, avatar_url, bio, university, faculty, grade, country, plan, created_at, updated_at, default_question_count;

-- name: UpdateUserQuestionSettings :one
UPDATE users SET
    default_question_count = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, firebase_uid, username, display_name, avatar_url, bio, university, faculty, grade, country, plan, created_at, updated_at, default_question_count;
