package api

import (
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/trashscanner/trashscanner_api/docs"
	"github.com/trashscanner/trashscanner_api/internal/api/middlewares"
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

	userRouter := root.PathPrefix("/users/me").Subrouter()
	userRouter.Use(s.authMiddleware, s.userMiddleware)
	userRouter.HandleFunc("", s.getUser).Methods(http.MethodGet)
	userRouter.HandleFunc("", s.deleteUser).Methods(http.MethodDelete)
	userRouter.HandleFunc("/logout", s.logout).Methods(http.MethodPost)
	userRouter.HandleFunc("/switch-password", s.switchPassword).Methods(http.MethodPut)
}
