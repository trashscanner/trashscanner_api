package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trashscanner/trashscanner_api/internal/api/dto"
)

var _ = Describe("User Flow E2E", func() {
	var client *http.Client
	var baseURL string

	BeforeEach(func() {
		jar, err := cookiejar.New(nil)
		Expect(err).NotTo(HaveOccurred())

		client = &http.Client{
			Jar: jar,
		}
		baseURL = tsServer.URL + "/api/v1"
	})

	It("Should register, login, get profile, upload avatar", func() {
		// 1. Register a new user
		regReqBody := dto.LoginUserRequest{
			Login:    "johndoe",
			Password: "secure_password123",
			Name:     "JohnDoe",
		}
		reqBytes, err := json.Marshal(regReqBody)
		Expect(err).NotTo(HaveOccurred())

		regResp, err := client.Post(baseURL+"/register", "application/json", bytes.NewReader(reqBytes))
		Expect(err).NotTo(HaveOccurred())

		bodyBytes, _ := io.ReadAll(regResp.Body)
		Expect(regResp.StatusCode).To(Equal(http.StatusCreated), "Register should succeed. Body: %s", string(bodyBytes))

		// Reset body so we can decode it
		regResp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		defer regResp.Body.Close()

		var authResp dto.AuthResponse
		err = json.NewDecoder(regResp.Body).Decode(&authResp)
		Expect(err).NotTo(HaveOccurred())
		Expect(authResp.User.Login).To(Equal("johndoe"))

		// 2. Login
		loginReqBody := dto.AuthRequest{
			Login:    "johndoe",
			Password: "secure_password123",
		}
		loginBytes, err := json.Marshal(loginReqBody)
		Expect(err).NotTo(HaveOccurred())

		loginResp, err := client.Post(baseURL+"/login", "application/json", bytes.NewReader(loginBytes))
		Expect(err).NotTo(HaveOccurred())
		Expect(loginResp.StatusCode).To(Equal(http.StatusOK), "Login should succeed")
		defer loginResp.Body.Close()

		// Verify cookies were set
		cookies := client.Jar.Cookies(loginResp.Request.URL)
		Expect(cookies).NotTo(BeEmpty(), "Cookies should be set by login")

		// 3. Get User Profile
		req, err := http.NewRequest(http.MethodGet, baseURL+"/users/me", nil)
		Expect(err).NotTo(HaveOccurred())

		meResp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(meResp.StatusCode).To(Equal(http.StatusOK), "Get /users/me should succeed")
		defer meResp.Body.Close()

		var meData dto.UserResponse
		err = json.NewDecoder(meResp.Body).Decode(&meData)
		Expect(err).NotTo(HaveOccurred())
		Expect(meData.Login).To(Equal("johndoe"))

		// 4. Upload Avatar
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreatePart(map[string][]string{
			"Content-Disposition": {`form-data; name="avatar"; filename="test_avatar.jpg"`},
			"Content-Type":        {"image/jpeg"},
		})
		Expect(err).NotTo(HaveOccurred())

		// Write a dummy image content
		_, err = part.Write([]byte("dummy jpeg image data"))
		Expect(err).NotTo(HaveOccurred())
		err = writer.Close()
		Expect(err).NotTo(HaveOccurred())

		avatarReq, err := http.NewRequest(http.MethodPut, baseURL+"/users/me/avatar", body)
		Expect(err).NotTo(HaveOccurred())
		avatarReq.Header.Set("Content-Type", writer.FormDataContentType())

		avatarResp, err := client.Do(avatarReq)
		Expect(err).NotTo(HaveOccurred())

		bodyBytes, _ = io.ReadAll(avatarResp.Body)
		Expect(avatarResp.StatusCode).To(Equal(http.StatusAccepted), "Upload avatar should be accepted. Body: %s", string(bodyBytes))

		// Reset body so we can decode it
		avatarResp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		defer avatarResp.Body.Close()

		var avatarUploadResp dto.UploadAvatarResponse
		err = json.NewDecoder(avatarResp.Body).Decode(&avatarUploadResp)
		Expect(err).NotTo(HaveOccurred())
		Expect(avatarUploadResp.AvatarURL).To(ContainSubstring("test_avatar.jpg"))
		Expect(avatarUploadResp.AvatarURL).NotTo(BeEmpty())

		// 5. Get User Profile again to check updated avatar
		reqMe2, err := http.NewRequest(http.MethodGet, baseURL+"/users/me", nil)
		Expect(err).NotTo(HaveOccurred())

		meResp2, err := client.Do(reqMe2)
		Expect(err).NotTo(HaveOccurred())
		Expect(meResp2.StatusCode).To(Equal(http.StatusOK), "Get /users/me again should succeed")
		defer meResp2.Body.Close()

		var meData2 dto.UserResponse
		err = json.NewDecoder(meResp2.Body).Decode(&meData2)
		Expect(err).NotTo(HaveOccurred())
		Expect(meData2.Avatar).NotTo(BeNil())
		Expect(*meData2.Avatar).To(Equal(avatarUploadResp.AvatarURL))
	})
})
