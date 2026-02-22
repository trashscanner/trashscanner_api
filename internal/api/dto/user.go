package dto

import (
	"github.com/trashscanner/trashscanner_api/internal/models"
)

type UserResponse models.User

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required,min=8,max=64"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=64,nefield=OldPassword"`
}

type UpdateUserRequest struct {
	Name string `json:"name" validate:"required,min=3,max=64,alphanum"`
}

type UploadAvatarRequest struct {
	Avatar string `json:"avatar" swaggertype:"string" format:"binary" example:"avatar.jpg" validate:"required"`
}

type UploadAvatarResponse struct {
	AvatarURL string `json:"avatar_url" example:"http://localhost:9000/trashscanner-images/user-id/avatars/avatar.jpg"`
}

type StatResponse models.Stat
