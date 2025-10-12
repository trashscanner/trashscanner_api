package dto

import (
	"github.com/trashscanner/trashscanner_api/internal/models"
)

type UserResponse models.User

type SwitchPasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required,min=8,max=64"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=64,nefield=OldPassword"`
}
