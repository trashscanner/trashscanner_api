package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/trashscanner/trashscanner_api/internal/api"
	"github.com/trashscanner/trashscanner_api/internal/auth"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/database"
	"github.com/trashscanner/trashscanner_api/internal/filestore"
	"github.com/trashscanner/trashscanner_api/internal/logging"
	"github.com/trashscanner/trashscanner_api/internal/store"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger := logging.NewLogger(cfg)
	logger.Infof("logger initialized with level %s", cfg.Log.Level)

	store, err := store.CreatePgStore(cfg)
	if err != nil {
		logger.Fatalf("failed to create store: %v", err)
	}

	if err := database.RunMigrations(store.Conn(), cfg); err != nil {
		logger.Errorf("failed to run migrations: %v", err)
		store.Close()
		return
	}

	fileStore, err := filestore.NewMinioStore(cfg)
	if err != nil {
		logger.Errorf("failed to create file store: %v", err)
		store.Close()
		return
	}

	auth, err := auth.NewJWTManager(cfg, store)
	if err != nil {
		logger.Errorf("failed to create auth manager: %v", err)
		store.Close()
		return
	}

	server := api.NewServer(cfg, store, fileStore, auth, logger)

	errCh := make(chan error, 1)
	signCh := make(chan os.Signal, 1)
	signal.Notify(signCh, os.Interrupt, syscall.SIGTERM)
	doneCh := make(chan struct{})

	go func() {
		err := server.Start()
		if !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}

		close(doneCh)
	}()

	for {
		select {
		case err := <-errCh:
			logger.Errorf("server error: %v", err)
			os.Exit(1)
		case <-doneCh:
			logger.Info("server stopped")
			os.Exit(0)
		case <-signCh:
			_ = server.Shutdown()
		}
	}
}
