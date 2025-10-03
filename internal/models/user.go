package models

import (
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
)

type User struct {
	ID             uuid.UUID
	Login          string
	HashedPassword string
	Avatar         url.URL
	Stat           *Stat
	Deleted        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
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

type LoginHistory db.LoginHistory
