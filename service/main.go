package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	store := NewPGStore(pool)
	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           NewServer(store),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}
