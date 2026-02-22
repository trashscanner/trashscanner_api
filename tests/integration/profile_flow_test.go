package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trashscanner/trashscanner_api/internal/api/dto"
)

var _ = Describe("Profile Flow E2E", func() {
	var client *http.Client
	var baseURL string

	BeforeEach(func() {
		jar, err := cookiejar.New(nil)
		Expect(err).NotTo(HaveOccurred())

		client = &http.Client{
			Jar: jar,
		}
		baseURL = tsServer.URL + "/api/v1"

		// 1. Prepare an authenticated user
		regReqBody := dto.LoginUserRequest{
			Login:    "profileuser",
			Password: "secure_password123",
			Name:     "ProfileUser",
		}
		reqBytes, _ := json.Marshal(regReqBody)
		regResp, err := client.Post(baseURL+"/register", "application/json", bytes.NewReader(reqBytes))
		Expect(err).NotTo(HaveOccurred())
		regResp.Body.Close()

		loginReqBody := dto.AuthRequest{
			Login:    "profileuser",
			Password: "secure_password123",
		}
		loginBytes, _ := json.Marshal(loginReqBody)
		loginResp, err := client.Post(baseURL+"/login", "application/json", bytes.NewReader(loginBytes))
		Expect(err).NotTo(HaveOccurred())
		Expect(loginResp.StatusCode).To(Equal(http.StatusOK), "Setup login should succeed")
		loginResp.Body.Close()
	})

	It("Should allow profile updates, password changes, token refresh, and logout", func() {
		// 1. Update Name via PATCH /users/me
		updateReqBody := dto.UpdateUserRequest{
			Name: "UpdatedProfileName",
		}
		updateBytes, _ := json.Marshal(updateReqBody)
		reqUpdate, _ := http.NewRequest(http.MethodPatch, baseURL+"/users/me", bytes.NewReader(updateBytes))
		reqUpdate.Header.Set("Content-Type", "application/json")

		updateResp, err := client.Do(reqUpdate)
		Expect(err).NotTo(HaveOccurred())
		Expect(updateResp.StatusCode).To(Equal(http.StatusOK), "Update profile should succeed")
		updateResp.Body.Close()

		// Verify name was updated
		reqMe, _ := http.NewRequest(http.MethodGet, baseURL+"/users/me", nil)
		meResp, err := client.Do(reqMe)
		Expect(err).NotTo(HaveOccurred())

		var meData dto.UserResponse
		_ = json.NewDecoder(meResp.Body).Decode(&meData)
		meResp.Body.Close()
		Expect(meData.Name).To(Equal("UpdatedProfileName"))

		// 2. Refresh Token via POST /refresh
		refreshReq, _ := http.NewRequest(http.MethodPost, baseURL+"/refresh", nil)
		refreshResp, err := client.Do(refreshReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(refreshResp.StatusCode).To(Equal(http.StatusAccepted), "Refresh should succeed")
		refreshResp.Body.Close()

		// 3. Change Password
		changePwdBody := dto.ChangePasswordRequest{
			OldPassword: "secure_password123",
			NewPassword: "new_secure_password123",
		}
		changePwdBytes, _ := json.Marshal(changePwdBody)
		reqChangePwd, _ := http.NewRequest(http.MethodPut, baseURL+"/users/me/change-password", bytes.NewReader(changePwdBytes))
		reqChangePwd.Header.Set("Content-Type", "application/json")

		changePwdResp, err := client.Do(reqChangePwd)
		Expect(err).NotTo(HaveOccurred())
		Expect(changePwdResp.StatusCode).To(Equal(http.StatusAccepted), "Change password should succeed")
		changePwdResp.Body.Close()

		// 4. Logout via POST /users/me/logout
		logoutReq, _ := http.NewRequest(http.MethodPost, baseURL+"/users/me/logout", nil)
		logoutResp, err := client.Do(logoutReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(logoutResp.StatusCode).To(Equal(http.StatusNoContent), "Logout should succeed")
		logoutResp.Body.Close()

		// 5. Verify the user is now unauthenticated
		reqMeAfterLogout, _ := http.NewRequest(http.MethodGet, baseURL+"/users/me", nil)
		meRespAfterLogout, err := client.Do(reqMeAfterLogout)
		Expect(err).NotTo(HaveOccurred())
		// NOTE: Because roleMiddleware runs *before* authMiddleware in router.go,
		// and test_config limits /api/v1/users/me/** to [user, admin],
		// an unauthenticated user (role "anonymous") hits a 403 Forbidden first
		// before it would hit a 401 Unauthorized in authMiddleware.
		Expect(meRespAfterLogout.StatusCode).To(Equal(http.StatusForbidden), "Should return 403 after logout due to roleMiddleware")
		meRespAfterLogout.Body.Close()

		// 6. Login with new password
		newLoginBody := dto.AuthRequest{
			Login:    "profileuser",
			Password: "new_secure_password123",
		}
		newLoginBytes, _ := json.Marshal(newLoginBody)
		newLoginResp, err := client.Post(baseURL+"/login", "application/json", bytes.NewReader(newLoginBytes))
		Expect(err).NotTo(HaveOccurred())
		Expect(newLoginResp.StatusCode).To(Equal(http.StatusOK), "Login with new password should succeed")
		newLoginResp.Body.Close()

		// 7. Delete User Profile
		reqDelete, _ := http.NewRequest(http.MethodDelete, baseURL+"/users/me", nil)
		deleteResp, err := client.Do(reqDelete)
		Expect(err).NotTo(HaveOccurred())
		Expect(deleteResp.StatusCode).To(Equal(http.StatusNoContent), "Delete user should succeed")
		deleteResp.Body.Close()

		// Verify deletion
		reqMeDeleted, _ := http.NewRequest(http.MethodGet, baseURL+"/users/me", nil)
		meRespDeleted, err := client.Do(reqMeDeleted)
		Expect(err).NotTo(HaveOccurred())
		// NOTE: Deleting the user does *not* clear their cookies in the client.
		// When meRespDeleted hits the server:
		// 1. softAuthMiddleware parses the token, finds the user ID in it, and attaches the (stale) role to the context
		// 2. roleMiddleware checks the role from context ("user"), and allows the request to /users/me
		// 3. authMiddleware validates the token signature, which passes (because tokens aren't instantly invalid unless explicitly blacklisted)
		// 4. Finally, userMiddleware attempts to fetch the user from the DB to refresh their data. This fails, returning ErrNotFound (404).
		Expect(meRespDeleted.StatusCode).To(Equal(http.StatusNotFound), "Deleted user should return not found")
		meRespDeleted.Body.Close()
	})
})
