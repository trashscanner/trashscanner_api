package api

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

const requestIDHeader = "X-Request-ID"

const (
	corsAllowMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	corsAllowHeaders = "Content-Type, Authorization, X-Request-ID"
)

func (s *Server) commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.setCORSHeaders(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

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

func (s *Server) setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", corsAllowMethods)
	w.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
	w.Header().Set("Access-Control-Max-Age", "3600")
}
