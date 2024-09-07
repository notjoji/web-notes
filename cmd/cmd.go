package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/notjoji/web-notes/internal/app"
	"github.com/notjoji/web-notes/internal/config"
	"github.com/notjoji/web-notes/internal/services"
	"github.com/notjoji/web-notes/pgdb"
)

func main() {
	if err := config.Load(".env"); err != nil {
		log.Fatal("Didn`t read .env config")
		return
	}

	ctx := context.Background()

	conn, err := pgdb.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatal("@[main] can't init service s3client: ", err)
		return
	}
	defer conn.Close()

	_ = app.NewApp(ctx, conn)
	// todo routing via application.Routes(httprouter)

	port := os.Getenv("API_PORT")
	server := &http.Server{
		Addr:        fmt.Sprintf(":%s", port),
		ReadTimeout: time.Second * 3,
		Handler:     http.HandlerFunc(services.Router),
	}
	if err := server.ListenAndServe(); err != nil {
		fmt.Println(fmt.Errorf("http listen err: %w", err))
	}
}
