package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/trashscanner/trashscanner_api/internal/models"
	testdata "github.com/trashscanner/trashscanner_api/internal/testdata"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func TestWriteLoginHistory_NoUser(t *testing.T) {
	server, storeMock, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", nil)

	server.writeLoginHistory(req, http.StatusOK, nil)

	storeMock.AssertNotCalled(t, "InsertLoginHistory", mock.Anything, mock.Anything)
}

func TestWriteLoginHistory_Success(t *testing.T) {
	server, storeMock, _, _ := newTestServer(t)

	user := testdata.User1
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", nil)
	req = req.WithContext(utils.SetUser(req.Context(), user))
	req.Header.Set("X-Real-IP", testdata.TestIPAddress.String())
	req.Header.Set("X-Location", testdata.TestLocation)
	req.Header.Set("User-Agent", testdata.TestUserAgent)

	storeMock.EXPECT().
		InsertLoginHistory(mock.Anything, mock.MatchedBy(func(history *models.LoginHistory) bool {
			assert.Equal(t, user.ID, history.UserID)
			assert.True(t, history.Success)
			require.NotNil(t, history.IpAddress)
			assert.Equal(t, testdata.TestIPAddress, *history.IpAddress)
			require.NotNil(t, history.UserAgent)
			assert.Equal(t, testdata.TestUserAgent, *history.UserAgent)
			require.NotNil(t, history.Location)
			assert.Equal(t, testdata.TestLocation, *history.Location)
			assert.Nil(t, history.FailureReason)
			return true
		})).
		Return(nil)

	server.writeLoginHistory(req, http.StatusOK, nil)
}

func TestWriteLoginHistory_WithFailure(t *testing.T) {
	server, storeMock, _, _ := newTestServer(t)

	user := testdata.User1
	ctx := utils.SetUser(context.Background(), user)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", nil).WithContext(ctx)
	req.Header.Set("X-Forwarded-For", "198.51.100.10, 203.0.113.5")
	req.RemoteAddr = "198.51.100.30:9999"
	failureErr := errors.New("invalid credentials")

	storeMock.EXPECT().
		InsertLoginHistory(mock.Anything, mock.MatchedBy(func(history *models.LoginHistory) bool {
			assert.Equal(t, user.ID, history.UserID)
			assert.False(t, history.Success)
			require.NotNil(t, history.IpAddress)
			expectedIP := netip.MustParseAddr("198.51.100.10")
			assert.Equal(t, expectedIP, *history.IpAddress)
			if assert.NotNil(t, history.FailureReason) {
				assert.Contains(t, *history.FailureReason, failureErr.Error())
			}
			return true
		})).
		Return(nil)

	server.writeLoginHistory(req, http.StatusUnauthorized, failureErr)
}
