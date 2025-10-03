package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
)

type UserStatus string

const (
	UserStatusNewbie      UserStatus = "newbie"          // Новичок
	UserStatusEcoScout    UserStatus = "eco_scout"       // Эко-разведчик
	UserStatusGreenGuard  UserStatus = "green_guard"     // Зеленый страж
	UserStatusEcoWarrior  UserStatus = "eco_warrior"     // Эко-воин
	UserStatusNatureHero  UserStatus = "nature_hero"     // Герой природы
	UserStatusEarthDefend UserStatus = "earth_defender"  // Защитник Земли
	UserStatusEcoChampion UserStatus = "eco_champion"    // Эко-чемпион
	UserStatusPlanetGuard UserStatus = "planet_guardian" // Хранитель планеты
	UserStatusEcoLegend   UserStatus = "eco_legend"      // Эко-легенда
)

func (s UserStatus) Valid() bool {
	switch s {
	case UserStatusNewbie, UserStatusEcoScout, UserStatusGreenGuard,
		UserStatusEcoWarrior, UserStatusNatureHero, UserStatusEarthDefend,
		UserStatusEcoChampion, UserStatusPlanetGuard, UserStatusEcoLegend:
		return true
	}
	return false
}

type TrashKind string

const (
	TrashKindPlastic TrashKind = "plastic"
	TrashKindPaper   TrashKind = "paper"
	TrashKindMetal   TrashKind = "metal"
	TrashKindGlass   TrashKind = "glass"
	TrashKindOrganic TrashKind = "organic"
	TrashKindOther   TrashKind = "other"
)

func (k TrashKind) Valid() bool {
	switch k {
	case TrashKindPlastic, TrashKindPaper, TrashKindMetal,
		TrashKindGlass, TrashKindOrganic, TrashKindOther:
		return true
	}
	return false
}

type Stat struct {
	ID            uuid.UUID
	Status        UserStatus
	Rating        int
	FilesScanned  int
	TotalWeight   float64
	LastScannedAt time.Time
	Achievements  []Achievement
	TrashByTypes  map[TrashKind]int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (s *Stat) Model(stat db.Stat) {
	s.ID = stat.ID
	s.Status = UserStatus(stat.Status)
	s.Rating = int(stat.Rating)
	s.FilesScanned = int(stat.FilesScanned)
	s.TotalWeight = stat.TotalWeight
	s.CreatedAt = stat.CreatedAt
	s.UpdatedAt = stat.UpdatedAt
}

type Achievement struct {
	// TODO
}
