package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"splitwise-assignment/api/internal/auth"
	"splitwise-assignment/api/internal/config"
	"splitwise-assignment/api/internal/httpapi"
	"splitwise-assignment/api/internal/importer"
	"splitwise-assignment/api/internal/store"
)

func main() {
	cfg := config.Load()
	memory := store.NewMemory()
	authStore := auth.Store(auth.DisabledStore{})

	if cfg.DatabaseURL != "" {
		pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("connect database: %v", err)
		}
		defer pool.Close()
		authStore = auth.NewPostgresStore(pool)
	}

	for _, path := range []string{"sample-data/expenses_export.csv", "../sample-data/expenses_export.csv"} {
		if file, err := os.Open(path); err == nil {
			if report, parseErr := importer.Parse(file); parseErr == nil {
				memory.ReplaceImport(report)
			}
			_ = file.Close()
			break
		}
	}

	server := httpapi.New(memory, authStore)
	log.Printf("api listening on %s", cfg.Addr)
	if cfg.DatabaseURL == "" {
		log.Print("DATABASE_URL not set; auth endpoints are disabled and expenses use in-memory storage")
	}
	if err := http.ListenAndServe(cfg.Addr, server); err != nil {
		log.Fatal(err)
	}
}
