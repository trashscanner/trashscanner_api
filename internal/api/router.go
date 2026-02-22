package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/trashscanner/trashscanner_api/docs"
)

const (
	predictionIDTag = "prediction_id"

	offsetQueryKey = "offset"
	limitQueryKey  = "limit"
	defaultLimit   = 100
	defaultOffset  = 0
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

	root.Use(mux.CORSMethodMiddleware(root), s.commonMiddleware, s.softAuthMiddleware, s.roleMiddleware)
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
	adminRouter.Use(s.authMiddleware)
	adminRouter.HandleFunc("/users", s.getUsersList).Methods(http.MethodGet)
	adminRouter.HandleFunc("/users", s.createUser).Methods(http.MethodPost)
}
