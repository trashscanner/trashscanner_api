package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/trashscanner/trashscanner_api/docs"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/rbac"
)

const (
	predictionIDTag  = "prediction_id"
	userIDTag        = "user_id"
	offsetQueryKey   = "offset"
	limitQueryKey    = "limit"
	defaultLimit     = 100
	defaultOffset    = 0
	corsAllowMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	corsAllowHeaders = "Content-Type, Authorization, X-Request-ID"
)

func (s *Server) initRouter() {
	s.router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	root := s.router.PathPrefix(apiPrefix).Subrouter().StrictSlash(true)
	root.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "endpoint not found", http.StatusNotFound)
	})
	root.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)

			origin := r.Header.Get("Origin")
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", corsAllowMethods)
				w.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
				w.Header().Set("Access-Control-Max-Age", "3600")
			}

			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
	root.Use(mux.CORSMethodMiddleware(root), s.commonMiddleware)
	root.HandleFunc("/health", s.healthCheck).Methods(http.MethodGet)
	root.HandleFunc("/refresh", s.refresh).Methods(http.MethodPost)

	authRouter := root.PathPrefix("").Subrouter()
	authRouter.Use(s.loginMiddleware)
	authRouter.HandleFunc("/login", s.login).Methods(http.MethodPost)
	authRouter.HandleFunc("/register", s.register).Methods(http.MethodPost)

	userRouter := root.PathPrefix("/users/me").Subrouter()
	userRouter.Use(s.authMiddleware, s.userMiddleware)
	userRouter.HandleFunc("", s.getUser).Methods(http.MethodGet)
	userRouter.HandleFunc("", s.updateUser).Methods(http.MethodPatch)
	userRouter.HandleFunc("", s.deleteUser).Methods(http.MethodDelete)
	userRouter.HandleFunc("/avatar", s.setAvatar).Methods(http.MethodPut)
	userRouter.HandleFunc("/avatar", s.deleteAvatar).Methods(http.MethodDelete)
	userRouter.HandleFunc("/logout", s.logout).Methods(http.MethodPost)
	userRouter.HandleFunc("/change-password", s.changePassword).Methods(http.MethodPut)

	predictionRouter := root.PathPrefix("/predictions").Subrouter()
	predictionRouter.Use(s.authMiddleware)
	predictionRouter.HandleFunc("", s.startPrediction).Methods(http.MethodPost)
	predictionRouter.HandleFunc("", s.listPredictions).Methods(http.MethodGet)
	predictionRouter.HandleFunc(fmt.Sprintf("/{%s}", predictionIDTag), s.getPrediction).Methods(http.MethodGet)

	adminRouter := root.PathPrefix("/admin").Subrouter()
	adminRouter.Use(s.authMiddleware, rbac.RequireRole(s.WriteError, models.RoleAdmin))
	adminRouter.HandleFunc("/users", s.getUsersList).Methods(http.MethodGet)
	adminRouter.HandleFunc("/users", s.createUser).Methods(http.MethodPost)
	adminRouter.HandleFunc(fmt.Sprintf("/users/{%s}", userIDTag), s.getAdminUser).Methods(http.MethodGet)
}
