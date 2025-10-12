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
}

// @title TrashScanner API
// @version 1.0
// @description This is a sample server TrashScanner API.

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
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
	}
}

func (s *Server) Start() error {
	s.initRouter()

	return s.s.ListenAndServe()
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.s.Shutdown(ctx)
}

func (s *Server) WriteResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if data == nil && status != http.StatusNoContent {
		data = map[string]string{"status": http.StatusText(status)}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(data)
	if err != nil {
		s.WriteError(w, errlocal.NewErrInternal("failed to encode response", err.Error(), nil))
	}
}

func (s *Server) WriteError(w http.ResponseWriter, err error) {
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
	}
}
