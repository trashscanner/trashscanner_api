package utils

import (
	"context"
)

type ContextKey string

const (
	userCtxKey     ContextKey = "user"
	requestBodyKey ContextKey = "request_body"
	requestIDKey   ContextKey = "request_id"
)

func SetUser(ctx context.Context, user any) context.Context {
	return context.WithValue(ctx, userCtxKey, user)
}

func GetUser(ctx context.Context) any {
	return ctx.Value(userCtxKey)
}

func SetRequestBody(ctx context.Context, body any) context.Context {
	return context.WithValue(ctx, requestBodyKey, body)
}

func GetRequestBody(ctx context.Context) any {
	return ctx.Value(requestBodyKey)
}

func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDKey).(string)
	return requestID, ok
}
