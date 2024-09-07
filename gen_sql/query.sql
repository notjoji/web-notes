-- name: GetPageableNotesByUserId :many
SELECT n.*
FROM notes n
WHERE user_id = $1
ORDER BY n.id
LIMIT $2 OFFSET $3;

-- name: GetNoteById :one
SELECT DISTINCT n.*
FROM notes n
WHERE n.id = $1;

-- name: CreateNote :one
INSERT INTO notes (user_id, name, description, deadline_at)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: UpdateNote :one
UPDATE notes
SET name         = $1,
    description  = $2,
    is_completed = $3,
    deadline_at  = $4
WHERE id = $5
RETURNING id;

-- name: GetUserByLoginAndPassword :one
SELECT DISTINCT u.*
FROM users u
WHERE u.login = $1 AND u.password = $2;

-- name: CreateUser :one
INSERT INTO users (login, password)
VALUES ($1, $2)
RETURNING id;