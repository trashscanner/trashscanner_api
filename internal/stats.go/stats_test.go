package stats

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/store/mocks"
	"github.com/trashscanner/trashscanner_api/internal/testdata"
)

func TestUpdateStats(t *testing.T) {
	ctx := context.Background()
	t.Run("success", func(t *testing.T) {
		user := testdata.User1
		currentStats := *user.Stat
		prediction := testdata.PredictionCompleted

		ms := mocks.NewStore(t)
		ms.EXPECT().GetUser(mock.Anything, user.ID, true).Return(&user, nil)

		ms.EXPECT().UpdateStats(mock.Anything, mock.AnythingOfType("*models.Stat")).
			Run(func(_ context.Context, updatedStat *models.Stat) {
				assert.Equal(t, currentStats.FilesScanned+1, updatedStat.FilesScanned)
				assert.Equal(t, time.Now().Minute(), updatedStat.LastScannedAt.Minute())
				assert.Equal(t, 1, updatedStat.TrashByTypes["metal"])
				assert.Equal(t, currentStats.Status, updatedStat.Status)
				assert.Equal(t, currentStats.Rating+10, updatedStat.Rating)
			}).Return(nil).Once()

		err := UpdateStats(ctx, ms, &prediction)
		assert.NoError(t, err)
	})

	t.Run("success with nil trash types", func(t *testing.T) {
		user := testdata.User1
		user.Stat.TrashByTypes = nil
		prediction := testdata.PredictionCompleted

		ms := mocks.NewStore(t)
		ms.EXPECT().GetUser(mock.Anything, user.ID, true).Return(&user, nil)
		ms.EXPECT().UpdateStats(mock.Anything, mock.MatchedBy(func(stat *models.Stat) bool {
			return stat.TrashByTypes != nil && stat.TrashByTypes["metal"] == 1
		})).Return(nil).Once()

		err := UpdateStats(ctx, ms, &prediction)
		assert.NoError(t, err)
	})

	t.Run("prediction with error - no rating increase", func(t *testing.T) {
		user := testdata.User1
		currentStats := *user.Stat
		prediction := testdata.PredictionCompleted
		prediction.Error = "prediction failed"
		prediction.Result = nil

		ms := mocks.NewStore(t)
		ms.EXPECT().GetUser(mock.Anything, user.ID, true).Return(&user, nil)
		ms.EXPECT().UpdateStats(mock.Anything, mock.MatchedBy(func(stat *models.Stat) bool {
			return stat.Rating == currentStats.Rating && // Рейтинг не изменился
				stat.FilesScanned == currentStats.FilesScanned+1 &&
				!stat.LastScannedAt.IsZero()
		})).Return(nil).Once()

		err := UpdateStats(ctx, ms, &prediction)
		assert.NoError(t, err)
	})

	t.Run("get user error", func(t *testing.T) {
		prediction := testdata.PredictionCompleted

		ms := mocks.NewStore(t)
		ms.EXPECT().GetUser(mock.Anything, prediction.UserID, true).
			Return(nil, errlocal.NewErrNotFound("user not found", "", nil))

		err := UpdateStats(ctx, ms, &prediction)
		assert.Error(t, err)
		var notFoundErr *errlocal.ErrNotFound
		assert.ErrorAs(t, err, &notFoundErr)
	})

	t.Run("update stats error", func(t *testing.T) {
		user := testdata.User1
		prediction := testdata.PredictionCompleted

		ms := mocks.NewStore(t)
		ms.EXPECT().GetUser(mock.Anything, user.ID, true).Return(&user, nil)
		ms.EXPECT().UpdateStats(mock.Anything, mock.Anything).
			Return(errlocal.NewErrInternal("db error", "", nil))

		err := UpdateStats(ctx, ms, &prediction)
		assert.Error(t, err)
		var internalErr *errlocal.ErrInternal
		assert.ErrorAs(t, err, &internalErr)
	})

	t.Run("multiple trash types", func(t *testing.T) {
		user := testdata.User1
		user.Stat.TrashByTypes = map[string]int{}
		prediction := testdata.PredictionCompleted
		prediction.Result = models.PredictionResult{
			"plastic":   0.40,
			"metal":     0.30,
			"cardboard": 0.20,
			"glass":     0.10,
		}

		ms := mocks.NewStore(t)
		ms.EXPECT().GetUser(mock.Anything, user.ID, true).Return(&user, nil)
		ms.EXPECT().UpdateStats(mock.Anything, mock.MatchedBy(func(stat *models.Stat) bool {
			return stat.TrashByTypes["plastic"] == 1 &&
				stat.TrashByTypes["metal"] == 1 &&
				stat.TrashByTypes["cardboard"] == 1 &&
				stat.TrashByTypes["glass"] == 1 &&
				len(stat.TrashByTypes) == 4
		})).Return(nil).Once()

		err := UpdateStats(ctx, ms, &prediction)
		assert.NoError(t, err)
	})

	t.Run("status upgrade", func(t *testing.T) {
		user := testdata.User1
		user.Stat.Rating = 95 // После +10 станет 105 -> eco_scout
		user.Stat.Status = models.UserStatusNewbie
		prediction := testdata.PredictionCompleted

		ms := mocks.NewStore(t)
		ms.EXPECT().GetUser(mock.Anything, user.ID, true).Return(&user, nil)
		ms.EXPECT().UpdateStats(mock.Anything, mock.MatchedBy(func(stat *models.Stat) bool {
			return stat.Rating == 105 && stat.Status == models.UserStatusEcoScout
		})).Return(nil).Once()

		err := UpdateStats(ctx, ms, &prediction)
		assert.NoError(t, err)
	})

	t.Run("last scanned at updated", func(t *testing.T) {
		user := testdata.User1
		oldTime := time.Now().Add(-24 * time.Hour)
		user.Stat.LastScannedAt = oldTime
		prediction := testdata.PredictionCompleted

		ms := mocks.NewStore(t)
		ms.EXPECT().GetUser(mock.Anything, user.ID, true).Return(&user, nil)
		ms.EXPECT().UpdateStats(mock.Anything, mock.MatchedBy(func(stat *models.Stat) bool {
			return stat.LastScannedAt.After(oldTime) &&
				time.Since(stat.LastScannedAt) < time.Second
		})).Return(nil).Once()

		err := UpdateStats(ctx, ms, &prediction)
		assert.NoError(t, err)
	})
}

func TestCalculateUserStatus(t *testing.T) {
	tests := []struct {
		name     string
		rating   int
		expected models.UserStatus
	}{
		{"newbie - rating 0", 0, models.UserStatusNewbie},
		{"newbie - rating 99", 99, models.UserStatusNewbie},
		{"eco_scout - rating 100", 100, models.UserStatusEcoScout},
		{"eco_scout - rating 299", 299, models.UserStatusEcoScout},
		{"green_guard - rating 300", 300, models.UserStatusGreenGuard},
		{"green_guard - rating 499", 499, models.UserStatusGreenGuard},
		{"eco_warrior - rating 500", 500, models.UserStatusEcoWarrior},
		{"eco_warrior - rating 999", 999, models.UserStatusEcoWarrior},
		{"nature_hero - rating 1000", 1000, models.UserStatusNatureHero},
		{"nature_hero - rating 1499", 1499, models.UserStatusNatureHero},
		{"earth_defender - rating 1500", 1500, models.UserStatusEarthDefend},
		{"earth_defender - rating 2999", 2999, models.UserStatusEarthDefend},
		{"eco_champion - rating 3000", 3000, models.UserStatusEcoChampion},
		{"eco_champion - rating 4999", 4999, models.UserStatusEcoChampion},
		{"planet_guardian - rating 5000", 5000, models.UserStatusPlanetGuard},
		{"planet_guardian - rating 9999", 9999, models.UserStatusPlanetGuard},
		{"eco_legend - rating 10000", 10000, models.UserStatusEcoLegend},
		{"eco_legend - rating 100000", 100000, models.UserStatusEcoLegend},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stat := &models.Stat{Rating: tt.rating}
			result := calculateUserStatus(stat)
			assert.Equal(t, tt.expected, result)
		})
	}
}
