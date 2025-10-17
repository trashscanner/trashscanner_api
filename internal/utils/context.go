package utils

import (
	"context"
	"time"

	"github.com/trashscanner/trashscanner_api/internal/models"
)

type ContextKey string

const (
	UserCtxKey     ContextKey = "user"
	RequestBodyKey ContextKey = "request-body"
	RequestIDKey   ContextKey = "request-id"
	TimeKey        ContextKey = "time"
	PathKey        ContextKey = "path"
	MethodKey      ContextKey = "method"
)

var ContextKeys = map[ContextKey]struct{}{
	UserCtxKey:     {},
	RequestBodyKey: {},
	RequestIDKey:   {},
	TimeKey:        {},
	PathKey:        {},
	MethodKey:      {},
}

func SetUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, UserCtxKey, user)
}

func GetUser(ctx context.Context) *models.User {
	return ctx.Value(UserCtxKey).(*models.User)
}

func SetRequestBody(ctx context.Context, body any) context.Context {
	return context.WithValue(ctx, RequestBodyKey, body)
}

func GetRequestBody(ctx context.Context) any {
	return ctx.Value(RequestBodyKey)
}

func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(RequestIDKey).(string)
	return requestID, ok
}

func GetPath(ctx context.Context) (string, bool) {
	path, ok := ctx.Value(PathKey).(string)
	return path, ok
}

func GetMethod(ctx context.Context) (string, bool) {
	method, ok := ctx.Value(MethodKey).(string)
	return method, ok
}

func GetContextValue(ctx context.Context, key ContextKey) (interface{}, bool) {
	val := ctx.Value(key)
	if val == nil {
		return nil, false
	}
	return val, true
}

func ElapsedTime(ctx context.Context) (time.Duration, bool) {
	startTime, ok := ctx.Value(TimeKey).(time.Time)
	if !ok {
		return 0, false
	}
	return time.Since(startTime), true
}
