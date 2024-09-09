// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: query.sql

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const CreateNote = `-- name: CreateNote :one
INSERT INTO notes (user_id, name, description, deadline_at)
VALUES ($1, $2, $3, $4)
RETURNING id
`

type CreateNoteParams struct {
	UserID      int64       `db:"user_id" json:"user_id"`
	Name        string      `db:"name" json:"name"`
	Description *string     `db:"description" json:"description"`
	DeadlineAt  pgtype.Date `db:"deadline_at" json:"deadline_at"`
}

func (q *Queries) CreateNote(ctx context.Context, arg CreateNoteParams) (int64, error) {
	row := q.db.QueryRow(ctx, CreateNote,
		arg.UserID,
		arg.Name,
		arg.Description,
		arg.DeadlineAt,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const CreateUser = `-- name: CreateUser :one
INSERT INTO users (login, password)
VALUES ($1, $2)
RETURNING id
`

type CreateUserParams struct {
	Login    string `db:"login" json:"login"`
	Password string `db:"password" json:"password"`
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (int64, error) {
	row := q.db.QueryRow(ctx, CreateUser, arg.Login, arg.Password)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const GetNoteById = `-- name: GetNoteById :one
SELECT DISTINCT n.id, n.user_id, n.name, n.description, n.is_completed, n.created_at, n.deadline_at
FROM notes n
WHERE n.id = $1
`

func (q *Queries) GetNoteById(ctx context.Context, id int64) (*Note, error) {
	row := q.db.QueryRow(ctx, GetNoteById, id)
	var i Note
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Name,
		&i.Description,
		&i.IsCompleted,
		&i.CreatedAt,
		&i.DeadlineAt,
	)
	return &i, err
}

const GetPageableNotesByUserId = `-- name: GetPageableNotesByUserId :many
SELECT n.id, n.user_id, n.name, n.description, n.is_completed, n.created_at, n.deadline_at
FROM notes n
WHERE user_id = $1
ORDER BY n.id
LIMIT $2 OFFSET $3
`

type GetPageableNotesByUserIdParams struct {
	UserID int64 `db:"user_id" json:"user_id"`
	Limit  int32 `db:"limit" json:"limit"`
	Offset int32 `db:"offset" json:"offset"`
}

func (q *Queries) GetPageableNotesByUserId(ctx context.Context, arg GetPageableNotesByUserIdParams) ([]*Note, error) {
	rows, err := q.db.Query(ctx, GetPageableNotesByUserId, arg.UserID, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*Note{}
	for rows.Next() {
		var i Note
		if err := rows.Scan(
			&i.ID,
			&i.UserID,
			&i.Name,
			&i.Description,
			&i.IsCompleted,
			&i.CreatedAt,
			&i.DeadlineAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const GetUserByLoginAndPassword = `-- name: GetUserByLoginAndPassword :one
SELECT DISTINCT u.id, u.login, u.password
FROM users u
WHERE u.login = $1 AND u.password = $2
`

type GetUserByLoginAndPasswordParams struct {
	Login    string `db:"login" json:"login"`
	Password string `db:"password" json:"password"`
}

func (q *Queries) GetUserByLoginAndPassword(ctx context.Context, arg GetUserByLoginAndPasswordParams) (*User, error) {
	row := q.db.QueryRow(ctx, GetUserByLoginAndPassword, arg.Login, arg.Password)
	var i User
	err := row.Scan(&i.ID, &i.Login, &i.Password)
	return &i, err
}

const UpdateNote = `-- name: UpdateNote :one
UPDATE notes
SET name         = $1,
    description  = $2,
    is_completed = $3,
    deadline_at  = $4
WHERE id = $5
RETURNING id
`

type UpdateNoteParams struct {
	Name        string      `db:"name" json:"name"`
	Description *string     `db:"description" json:"description"`
	IsCompleted bool        `db:"is_completed" json:"is_completed"`
	DeadlineAt  pgtype.Date `db:"deadline_at" json:"deadline_at"`
	ID          int64       `db:"id" json:"id"`
}

func (q *Queries) UpdateNote(ctx context.Context, arg UpdateNoteParams) (int64, error) {
	row := q.db.QueryRow(ctx, UpdateNote,
		arg.Name,
		arg.Description,
		arg.IsCompleted,
		arg.DeadlineAt,
		arg.ID,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}
