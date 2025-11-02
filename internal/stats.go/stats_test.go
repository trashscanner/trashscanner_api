package stats

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
}
