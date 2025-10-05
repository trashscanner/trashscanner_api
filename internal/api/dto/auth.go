package dto

import (
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

type AuthRequest struct {
	Login    string `json:"login" validate:"required,min=3,max=32,alphanum"`
	Password string `json:"password" validate:"required,min=8,max=64"`
}

func (r *AuthRequest) ToModel() models.User {
	hp, _ := utils.HashPass(r.Password)
	return models.User{
		Login:          r.Login,
		HashedPassword: hp,
	}
}

type AuthResponse struct {
	User struct {
		ID    string `json:"id"`
		Login string `json:"login"`
	} `json:"user"`
	Tokens Tokens `json:"tokens"`
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func NewAuthResponse(user models.User, access, refresh string) AuthResponse {
	return AuthResponse{
		User: struct {
			ID    string `json:"id"`
			Login string `json:"login"`
		}{
			ID:    user.ID.String(),
			Login: user.Login,
		},
		Tokens: Tokens{
			AccessToken:  access,
			RefreshToken: refresh,
		},
	}
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshResponse struct {
	Tokens Tokens `json:"tokens"`
}

func NewRefreshResponse(access, refresh string) RefreshResponse {
	return RefreshResponse{
		Tokens: Tokens{
			AccessToken:  access,
			RefreshToken: refresh,
		},
	}
}
