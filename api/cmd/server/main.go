package main

import (
	"log"
	"net/http"
	"os"

	"splitwise-assignment/api/internal/config"
	"splitwise-assignment/api/internal/httpapi"
	"splitwise-assignment/api/internal/importer"
	"splitwise-assignment/api/internal/store"
)

func main() {
	cfg := config.Load()
	memory := store.NewMemory()

	for _, path := range []string{"sample-data/expenses_export.csv", "../sample-data/expenses_export.csv"} {
		if file, err := os.Open(path); err == nil {
			if report, parseErr := importer.Parse(file); parseErr == nil {
				memory.ReplaceImport(report)
			}
			_ = file.Close()
			break
		}
	}

	server := httpapi.New(memory)
	log.Printf("api listening on %s", cfg.Addr)
	if cfg.DatabaseURL == "" {
		log.Print("DATABASE_URL not set; using in-memory store for local demo")
	}
	if err := http.ListenAndServe(cfg.Addr, server); err != nil {
		log.Fatal(err)
	}
}
