package models

import (
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
)

type User struct {
	ID             uuid.UUID `json:"id"`
	Login          string    `json:"login"`
	HashedPassword string    `json:"-"`
	Avatar         url.URL   `json:"avatar,omitempty" swaggertype:"string"`
	Stat           *Stat     `json:"stat,omitempty"`
	Deleted        bool      `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (u *User) Model(user db.User) {
	u.ID = user.ID
	u.Login = user.Login
	u.HashedPassword = user.HashedPassword
	if user.Avatar != nil {
		avatar, err := url.Parse(*user.Avatar)
		if err == nil {
			u.Avatar = *avatar
		}
	}
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
