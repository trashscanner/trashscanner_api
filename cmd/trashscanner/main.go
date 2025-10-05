package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/trashscanner/trashscanner_api/internal/api"
	"github.com/trashscanner/trashscanner_api/internal/auth"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/database"
	"github.com/trashscanner/trashscanner_api/internal/store"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	store, err := store.NewPGStore(cfg)
	if err != nil {
		log.Fatalf("failed to create store: %v", err)
	}

	if err := database.RunMigrations(store.Conn(), cfg); err != nil {
		store.Close()
		log.Fatalf("failed to run migrations: %v", err)
	}

	auth, err := auth.NewJWTManager(cfg, store)
	if err != nil {
		store.Close()
		log.Fatalf("failed to create auth manager: %v", err)
	}

	server := api.NewServer(cfg, store, auth)

	go func() {
		if err := server.Start(); err != nil {
			store.Close()
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	signCh := make(chan os.Signal, 1)
	signal.Notify(signCh, os.Interrupt, syscall.SIGTERM)
	<-signCh

	log.Println("shutting down gracefully...")
	if err := server.Shutdown(); err != nil {
		log.Printf("error during shutdown: %v", err)
	}
}
