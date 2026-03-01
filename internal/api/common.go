package api

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

const requestIDHeader = "X-Request-ID"

const ()

func (s *Server) commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(requestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		w.Header().Set(requestIDHeader, requestID)
		r.Header.Set(requestIDHeader, requestID)

		ctx := r.Context()
		ctx = context.WithValue(ctx, utils.TimeKey, time.Now())
		ctx = context.WithValue(ctx, utils.PathKey, r.URL.Path)
		ctx = context.WithValue(ctx, utils.MethodKey, r.Method)
		ctx = context.WithValue(ctx, utils.RequestIDKey, requestID)
		r = r.WithContext(ctx)

		s.logger.WithContext(ctx).Info("handling request")

		next.ServeHTTP(w, r)

		elapsed, ok := utils.ElapsedTime(ctx)
		l := s.logger.WithContext(ctx)
		if ok {
			l = l.WithField("elapsed_ms", elapsed.Milliseconds())
		}
		l.Info("finished handling request")
	})
}
