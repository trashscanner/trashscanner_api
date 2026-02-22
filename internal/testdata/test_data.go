package testdata

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

var (
	NewUser = models.User{
		Name:           "test_name",
		Login:          "test_user",
		HashedPassword: "hashed_password",
		Role:           models.RoleUser,
	}
	User1ID = uuid.MustParse("04ae379a-31a6-4b32-a6d7-f6cdd844f81a")
	User1   = models.User{
		ID:             User1ID,
		Name:           "test_name_1",
		Login:          "test_user_1",
		HashedPassword: "hashed_password_1",
		Role:           models.RoleUser,
		Deleted:        false,
		Stat:           &Stats1,
	}
	User2ID = uuid.MustParse("13ae379a-31a6-4b32-a6d7-f6cdd844f82b")
	User2   = models.User{
		ID:             User2ID,
		Name:           "test_name_2",
		Login:          "test_user_2",
		HashedPassword: "hashed_password_2",
		Role:           models.RoleUser,
		Deleted:        false,
		Stat:           &Stats2,
	}

	Stats1ID = uuid.MustParse("24ae379a-31a6-4b32-a6d7-f6cdd844f83c")
	Stats1   = models.Stat{
		ID:           Stats1ID,
		Status:       models.UserStatusNatureHero,
		Rating:       1100,
		FilesScanned: 15,
		TrashByTypes: map[string]int{
			"plastic": 10,
			"paper":   5,
		},
	}
	Stats2ID = uuid.MustParse("34ae379a-31a6-4b32-a6d7-f6cdd844f84d")
	Stats2   = models.Stat{
		ID:     Stats2ID,
		Status: models.UserStatusNewbie,
		Rating: 100,
	}

	DeletedUserID = uuid.MustParse("44ae379a-31a6-4b32-a6d7-f6cdd844f85e")
	DeletedUser   = models.User{
		ID:             DeletedUserID,
		Login:          "deleted_user",
		HashedPassword: "hashed_password_deleted",
		Deleted:        true,
	}

	AvatarURL    = "https://example.com/avatar.jpg"
	NewAvatarURL = "https://example.com/new_avatar.png"

	NewHashedPassword = "new_hashed_password_123"

	LoginHistory1ID = uuid.MustParse("54ae379a-31a6-4b32-a6d7-f6cdd844f86f")
	LoginHistory1   = models.LoginHistory{
		ID:        LoginHistory1ID,
		UserID:    User1ID,
		Success:   true,
		IpAddress: addrPtr("192.168.1.1"),
		UserAgent: stringPtr("Mozilla/5.0"),
		Location:  stringPtr("Moscow, Russia"),
	}

	LoginHistory2ID = uuid.MustParse("64ae379a-31a6-4b32-a6d7-f6cdd844f87a")
	LoginHistory2   = models.LoginHistory{
		ID:            LoginHistory2ID,
		UserID:        User1ID,
		Success:       false,
		FailureReason: stringPtr("Invalid password"),
		IpAddress:     addrPtr("192.168.1.1"),
		UserAgent:     stringPtr("Mozilla/5.0"),
		Location:      stringPtr("Moscow, Russia"),
	}

	LoginHistory3ID = uuid.MustParse("74ae379a-31a6-4b32-a6d7-f6cdd844f88b")
	LoginHistory3   = models.LoginHistory{
		ID:        LoginHistory3ID,
		UserID:    User2ID,
		Success:   true,
		IpAddress: addrPtr("10.0.0.1"),
		UserAgent: stringPtr("Chrome/120.0"),
		Location:  stringPtr("Saint Petersburg, Russia"),
	}

	TestIPAddress = netip.MustParseAddr("203.0.113.42")
	TestUserAgent = "TestBot/1.0"
	TestLocation  = "Test City, Test Country"

	DBLoginHistory1 = db.LoginHistory{
		ID:        LoginHistory1ID,
		UserID:    User1ID,
		Success:   true,
		IpAddress: addrPtr("192.168.1.1"),
		UserAgent: stringPtr("Mozilla/5.0"),
		Location:  stringPtr("Moscow, Russia"),
		CreatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	DBLoginHistory2 = db.LoginHistory{
		ID:            LoginHistory2ID,
		UserID:        User1ID,
		Success:       false,
		FailureReason: stringPtr("Invalid password"),
		IpAddress:     addrPtr("192.168.1.1"),
		UserAgent:     stringPtr("Mozilla/5.0"),
		Location:      stringPtr("Moscow, Russia"),
		CreatedAt:     time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
	}

	RefreshToken1ID = uuid.MustParse("a4ae379a-31a6-4b32-a6d7-f6cdd844f91e")
	RefreshToken1   = models.RefreshToken{
		ID:        RefreshToken1ID,
		UserID:    User1ID,
		TokenHash: "hash_token_1",
		ExpiresAt: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		Revoked:   false,
	}

	RefreshToken2ID = uuid.MustParse("b4ae379a-31a6-4b32-a6d7-f6cdd844f92f")
	RefreshToken2   = models.RefreshToken{
		ID:        RefreshToken2ID,
		UserID:    User1ID,
		TokenHash: "hash_token_2",
		ExpiresAt: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		Revoked:   false,
	}

	RefreshToken3ID = uuid.MustParse("c4ae379a-31a6-4b32-a6d7-f6cdd844f93a")
	RefreshToken3   = models.RefreshToken{
		ID:        RefreshToken3ID,
		UserID:    User2ID,
		TokenHash: "hash_token_3",
		ExpiresAt: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		Revoked:   false,
	}

	RevokedTokenID = uuid.MustParse("d4ae379a-31a6-4b32-a6d7-f6cdd844f94b")
	RevokedToken   = models.RefreshToken{
		ID:        RevokedTokenID,
		UserID:    User1ID,
		TokenHash: "hash_revoked_token",
		ExpiresAt: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		Revoked:   true,
	}

	ScanID        = uuid.MustParse("a34d8ee6-fe5f-4795-ba7e-6e127ec2aa02")
	ScanURL       = User1ID.String() + "/scans/" + ScanID.String()
	PredictionID  = uuid.MustParse("00e0cc2f-47e0-4e0d-a0f5-401fe9f0f5d6")
	NewPrediction = models.Prediction{
		UserID:    User1ID,
		TrashScan: ScanURL,
		Status:    models.PredictionProcessingStatus,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	PredictionCompleted = models.Prediction{
		UserID:    User1ID,
		TrashScan: ScanURL,
		Status:    models.PredictionCompletedStatus,
		Result:    models.PredictionResult{models.TrashTypeMetal: 0.9},
	}
)

func stringPtr(s string) *string {
	return &s
}

func addrPtr(s string) *netip.Addr {
	addr := netip.MustParseAddr(s)
	return &addr
}
