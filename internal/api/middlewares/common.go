package middlewares

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func Common(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		w.Header().Set("X-Request-ID", requestID)
		r.Header.Set("X-Request-ID", requestID)

		utils.SetRequestID(r.Context(), requestID)

		next.ServeHTTP(w, r.WithContext(r.Context()))
		// TODO logs
	})
}
