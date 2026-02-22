package dto

import (
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

type AuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginUserRequest struct {
	Login    string `json:"login" validate:"required,min=3,max=32,alphanum"`
	Password string `json:"password" validate:"required,min=8,max=64"`
	Name     string `json:"name,omitempty" validate:"omitempty,min=3,max=64,alphanum"`
}

func (r *LoginUserRequest) ToModel() models.User {
	hp, _ := utils.HashPass(r.Password)
	return models.User{
		Name:           r.Name,
		Login:          r.Login,
		HashedPassword: hp,
		Role:           models.RoleUser,
	}
}

type AuthResponse struct {
	User struct {
		ID    string `json:"id"`
		Login string `json:"login"`
	} `json:"user"`
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
	}
}
