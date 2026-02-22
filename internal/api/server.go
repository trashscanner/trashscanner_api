package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/trashscanner/trashscanner_api/docs"
	"github.com/trashscanner/trashscanner_api/internal/auth"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/filestore"
	"github.com/trashscanner/trashscanner_api/internal/logging"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/store"
)

const (
	defaultHost    = "0.0.0.0"
	defaultPort    = "8080"
	defaultTimeout = time.Second * 10
	apiPrefix      = "/api/v1"
)

type Server struct {
	s           *http.Server
	router      *mux.Router
	store       store.Store
	fileStore   filestore.FileStore
	authManager auth.AuthManager
	authConfig  *config.AuthConfig
	predictor   predictor
	logger      *logging.Logger
	healthy     bool
}

type predictor interface {
	Predict(ctx context.Context, scanURL string) (*models.Prediction, error)
}

// @title TrashScanner API
// @version 1.0
// @description This is a sample server TrashScanner API.

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host 0.0.0.0:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func NewServer(
	cfg config.Config,
	store store.Store,
	fileStore filestore.FileStore,
	authManager auth.AuthManager,
	predictor predictor,
	logger *logging.Logger,
) *Server {
	r := mux.NewRouter()

	return &Server{
		s: &http.Server{
			Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
			Handler:      r,
			WriteTimeout: defaultTimeout,
			ReadTimeout:  defaultTimeout,
		},
		router:      r,
		store:       store,
		fileStore:   fileStore,
		authManager: authManager,
		authConfig:  &cfg.AuthConfig,
		predictor:   predictor,
		logger:      logger.WithApiTag(),
	}
}

func (s *Server) InitRouter() *mux.Router {
	s.initRouter()
	return s.router
}

func (s *Server) Start() error {
	s.logger.Infof("starting server at %s", s.s.Addr)
	s.initRouter()
	s.healthy = true

	return s.s.ListenAndServe()
}

func (s *Server) Shutdown() error {
	s.logger.Infof("shutting down server at %s", s.s.Addr)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.healthy = false

	if err := s.s.Shutdown(ctx); err != nil {
		s.logger.Warnf("graceful shutdown failed, forcing close: %v", err)
		return s.s.Close()
	}

	return nil
}

func (s *Server) WriteResponse(w http.ResponseWriter, r *http.Request, status int, data any) {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		s.logger.WithContext(r.Context()).WithField("status", status).Info("request processed")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data == nil {
		data = map[string]string{"status": http.StatusText(status)}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(data)
	if err != nil {
		s.WriteError(w, r, errlocal.NewErrInternal("failed to encode response", err.Error(), nil))
		return
	}

	s.logger.WithContext(r.Context()).WithField("status", status).Info("request processed")
}

func (s *Server) WriteError(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	var errLocal errlocal.LocalError
	if !errors.As(err, &errLocal) {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(errLocal.Code())
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if encodeErr := encoder.Encode(err); encodeErr != nil {
		http.Error(w, `{"message":"failed to encode error response"}`, http.StatusInternalServerError)
		return
	}

	s.logger.WithContext(r.Context()).WithError(err).Error("request processed with error")
}

// HealthCheck godoc
// @Summary Health check
// @Description Check server health
// @Tags health
// @Produce json
// @Success 200 {object} bool "Is server healthy"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Router /health [get]
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	s.WriteResponse(w, r, http.StatusOK, s.healthy)
}
