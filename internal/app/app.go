package app

import (
	"context"
	"github.com/notjoji/web-notes/internal/repository"
	"github.com/notjoji/web-notes/pgdb"
)

type App struct {
	ctx   context.Context
	conn  *pgdb.Connection
	cache map[string]*repository.User
}

func NewApp(ctx context.Context, conn *pgdb.Connection) *App {
	return &App{ctx, conn, make(map[string]*repository.User)}
}
