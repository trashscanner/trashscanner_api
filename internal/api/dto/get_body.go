package dto

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type HTTPResource interface {
	AuthRequest | RefreshRequest | SwitchPasswordRequest
}

func GetRequestBody[T any](r *http.Request) (*T, error) {
	var body T
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}
	err := validator.New().Struct(body)
	if err != nil {
		return nil, err
	}

	return &body, nil
}
