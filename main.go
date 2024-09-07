package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/notjoji/web-notes/internal/app"
	"github.com/notjoji/web-notes/internal/config"
	"github.com/notjoji/web-notes/internal/repository"
	"github.com/notjoji/web-notes/pgdb"
)

func main() {
	if err := config.Load(".env"); err != nil {
		log.Fatal("Can't load .env config, closing...")
		return
	}

	ctx := context.Background()

	conn, err := pgdb.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatal("Can't init database connect: ", err)
		return
	}
	db := repository.New(conn.Pool())
	defer conn.Close()

	application := app.NewApp(ctx, db)
	router := httprouter.New()
	application.Routes(router)

	port := os.Getenv("API_PORT")
	server := &http.Server{
		Addr:        fmt.Sprintf(":%s", port),
		ReadTimeout: time.Second * 3,
		Handler:     router,
	}
	if err := server.ListenAndServe(); err != nil {
		fmt.Println(fmt.Errorf("http listen err: %w", err))
	}
}
