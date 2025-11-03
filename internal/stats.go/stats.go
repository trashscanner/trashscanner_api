package stats

import (
	"context"
	"time"

	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/store"
)

func UpdateStats(
	ctx context.Context,
	store store.Store,
	newPrediction *models.Prediction,
) error {
	user, err := store.GetUser(ctx, newPrediction.UserID, true)
	if err != nil {
		return err
	}

	currentStats := user.Stat
	if currentStats.TrashByTypes == nil {
		currentStats.TrashByTypes = make(map[string]int)
	}

	currentStats.FilesScanned++
	currentStats.LastScannedAt = time.Now()

	if newPrediction.Error == "" {
		currentStats.Rating += 10

		for k := range newPrediction.Result {
			currentStats.TrashByTypes[k] += 1
		}
		currentStats.Status = calculateUserStatus(currentStats)
	}

	return store.UpdateStats(ctx, currentStats)
}

func calculateUserStatus(stat *models.Stat) models.UserStatus {
	if stat.Rating >= 10000 {
		return models.UserStatusEcoLegend
	}
	if stat.Rating >= 5000 {
		return models.UserStatusPlanetGuard
	}
	if stat.Rating >= 3000 {
		return models.UserStatusEcoChampion
	}
	if stat.Rating >= 1500 {
		return models.UserStatusEarthDefend
	}
	if stat.Rating >= 1000 {
		return models.UserStatusNatureHero
	}
	if stat.Rating >= 500 {
		return models.UserStatusEcoWarrior
	}
	if stat.Rating >= 300 {
		return models.UserStatusGreenGuard
	}
	if stat.Rating >= 100 {
		return models.UserStatusEcoScout
	}
	return models.UserStatusNewbie
}
