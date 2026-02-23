package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

type AdminUserResponse struct {
	ID        uuid.UUID   `json:"id"`
	Login     string      `json:"login"`
	Name      string      `json:"name"`
	Role      models.Role `json:"role"`
	Avatar    *string     `json:"avatar,omitempty"`
	Deleted   bool        `json:"deleted"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`

	LastLoginAt *time.Time    `json:"last_login_at,omitempty"`
	Stat        *StatResponse `json:"stat,omitempty"`
}

type AdminUserListResponse struct {
	TotalCount int64               `json:"total_count"`
	Limit      int                 `json:"limit"`
	Offset     int                 `json:"offset"`
	Users      []AdminUserResponse `json:"users"`
}

type CreateAdminRequest struct {
	Name     string      `json:"name" validate:"required,min=2,max=100"`
	Login    string      `json:"login" validate:"required,alphanum,min=3,max=50"`
	Password string      `json:"password" validate:"required,min=8"`
	Role     models.Role `json:"role" validate:"required"`
}

type AdminUserParams struct {
	Limit  int `query:"limit" validate:"min=1,max=100"`
	Offset int `query:"offset" validate:"min=0"`
}

func NewAdminUserListResponse(users []models.User, totalCount int64, limit, offset int) AdminUserListResponse {
	res := AdminUserListResponse{
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
		Users:      make([]AdminUserResponse, 0, len(users)),
	}

	for _, u := range users {
		res.Users = append(res.Users, NewAdminUserResponse(u))
	}

	return res
}

func NewAdminUserResponse(user models.User) AdminUserResponse {
	var stat *StatResponse
	if user.Stat != nil {
		s := StatResponse(*user.Stat)
		stat = &s
	}

	return AdminUserResponse{
		ID:          user.ID,
		Login:       user.Login,
		Name:        user.Name,
		Role:        user.Role,
		Avatar:      user.Avatar,
		Deleted:     user.Deleted,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		LastLoginAt: user.LastLoginAt,
		Stat:        stat,
	}
}

type AdminUserDetailResponse struct {
	AdminUserResponse
	Predictions []PredictionResponse `json:"predictions"`
	Limit       int                  `json:"limit"`
	Offset      int                  `json:"offset"`
}

func NewAdminUserDetailResponse(user models.User, predictions []*models.Prediction, limit, offset int) AdminUserDetailResponse {
	preds := make([]PredictionResponse, 0, len(predictions))
	for _, p := range predictions {
		preds = append(preds, PredictionResponse(*p))
	}

	return AdminUserDetailResponse{
		AdminUserResponse: NewAdminUserResponse(user),
		Predictions:       preds,
		Limit:             limit,
		Offset:            offset,
	}
}

func (req *CreateAdminRequest) ToModel() *models.User {
	hp, _ := utils.HashPass(req.Password)
	return &models.User{
		Name:           req.Name,
		Login:          req.Login,
		HashedPassword: hp,
		Role:           req.Role,
	}
}
