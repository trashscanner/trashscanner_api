package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
)

type User struct {
	ID             uuid.UUID `json:"id"`
	Login          string    `json:"login"`
	Name           string    `json:"name"`
	HashedPassword string    `json:"-"`
	Role           Role      `json:"role"`
	Avatar         *string   `json:"avatar,omitempty"`
	Stat           *Stat     `json:"stat,omitempty"`
	Deleted        bool      `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (u *User) Model(user db.User) {
	u.ID = user.ID
	u.Name = user.Name
	u.Login = user.Login
	u.HashedPassword = user.HashedPassword
	u.Role = Role(user.Role)
	u.Avatar = user.Avatar
	u.Deleted = user.Deleted
	u.CreatedAt = user.CreatedAt
	u.UpdatedAt = user.UpdatedAt
}

func (u *User) WithStat(stat db.Stat) {
	if u.Stat == nil {
		u.Stat = &Stat{}
	}
	u.Stat.Model(stat)
}

type RefreshToken db.RefreshToken

func NewRefreshFromClaims(hash string, claims jwt.RegisteredClaims) *RefreshToken {
	return &RefreshToken{
		UserID:    uuid.MustParse(claims.Subject),
		ExpiresAt: claims.ExpiresAt.Time,
		TokenHash: hash,
	}
}

type LoginHistory db.LoginHistory
