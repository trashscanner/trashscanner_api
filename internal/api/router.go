package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/trashscanner/trashscanner_api/docs"
	"github.com/trashscanner/trashscanner_api/internal/api/middlewares"
)

const (
	userKey = "user-id"
)

func (s *Server) initRouter() {
	s.router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	root := s.router.PathPrefix(apiPrefix).Subrouter().StrictSlash(true)
	root.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "endpoint not found", http.StatusNotFound)
	})
	root.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	root.Use(mux.CORSMethodMiddleware(root), middlewares.Common)

	loginRouter := root.PathPrefix("/login").Subrouter()
	loginRouter.Use(s.loginMiddleware)
	loginRouter.HandleFunc("", s.login).Methods(http.MethodPost)

	root.HandleFunc("/refresh", s.refresh).Methods(http.MethodPost)

	userRouter := root.PathPrefix("/users").Subrouter()
	userRouter.Use(s.authMiddleware, s.userMiddleware)
	userRouter.HandleFunc(fmt.Sprintf("/{%s}", userKey), s.getUser).Methods(http.MethodGet)
}
