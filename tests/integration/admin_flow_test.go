package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trashscanner/trashscanner_api/internal/api/dto"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

var _ = Describe("Admin Flow E2E", func() {
	var client *http.Client
	var baseURL string
	var adminUser models.User

	BeforeEach(func() {
		jar, err := cookiejar.New(nil)
		Expect(err).NotTo(HaveOccurred())

		client = &http.Client{
			Jar: jar,
		}
		baseURL = tsServer.URL + "/api/v1"

		// 1. Manually create an Admin User via store to bypass API endpoint restrictions
		hashedPass, _ := utils.HashPass("admin_secure_123")
		adminUser = models.User{
			ID:             uuid.New(),
			Login:          "superadmin",
			Name:           "Super Admin",
			HashedPassword: hashedPass,
			Role:           models.RoleAdmin,
		}
		err = pgStore.CreateUser(context.Background(), &adminUser)
		Expect(err).NotTo(HaveOccurred())

		// Login as Admin
		loginReqBody := dto.AuthRequest{
			Login:    "superadmin",
			Password: "admin_secure_123",
		}
		loginBytes, _ := json.Marshal(loginReqBody)
		loginResp, err := client.Post(baseURL+"/login", "application/json", bytes.NewReader(loginBytes))
		Expect(err).NotTo(HaveOccurred())
		Expect(loginResp.StatusCode).To(Equal(http.StatusOK), "Admin login should succeed")
		loginResp.Body.Close()
	})

	It("Should manage users via admin endpoints and enforce RBAC", func() {
		// 1. Admin gets user list
		reqList, _ := http.NewRequest(http.MethodGet, baseURL+"/admin/users", nil)
		listResp, err := client.Do(reqList)
		Expect(err).NotTo(HaveOccurred())

		bodyBytes, _ := io.ReadAll(listResp.Body)
		Expect(listResp.StatusCode).To(Equal(http.StatusOK), "Admin should be able to get users list. Body: %s", string(bodyBytes))
		listResp.Body.Close()

		var usersList dto.AdminUserListResponse
		err = json.Unmarshal(bodyBytes, &usersList)
		Expect(err).NotTo(HaveOccurred())
		Expect(usersList.TotalCount).To(BeNumerically(">", 0), "Should see at least the admin user")

		// 2. Admin creates a new user directly
		createReqBody := dto.CreateAdminRequest{
			Login:    "newuserbyadmin",
			Password: "user_secure_123",
			Name:     "NewUserCreatedByAdmin",
			Role:     models.RoleUser,
		}
		createBytes, _ := json.Marshal(createReqBody)
		reqCreate, _ := http.NewRequest(http.MethodPost, baseURL+"/admin/users", bytes.NewReader(createBytes))
		reqCreate.Header.Set("Content-Type", "application/json")

		createResp, err := client.Do(reqCreate)
		Expect(err).NotTo(HaveOccurred())
		Expect(createResp.StatusCode).To(Equal(http.StatusCreated), "Admin should be able to create users")
		createResp.Body.Close()

		// Verify the new user can log in
		newUserClient := &http.Client{}
		newLoginBody := dto.AuthRequest{
			Login:    "newuserbyadmin",
			Password: "user_secure_123",
		}
		newLoginBytes, _ := json.Marshal(newLoginBody)
		newLoginResp, err := newUserClient.Post(baseURL+"/login", "application/json", bytes.NewReader(newLoginBytes))
		Expect(err).NotTo(HaveOccurred())
		Expect(newLoginResp.StatusCode).To(Equal(http.StatusOK), "Newly created user should be able to log in")
		newLoginResp.Body.Close()

		// 3. Normal user tries to access admin routes (RBAC check)
		jar, _ := cookiejar.New(nil)
		newUserClient.Jar = jar
		newLoginResp, _ = newUserClient.Post(baseURL+"/login", "application/json", bytes.NewReader(newLoginBytes))
		newLoginResp.Body.Close() // Ensure cookies are stored

		reqFailList, _ := http.NewRequest(http.MethodGet, baseURL+"/admin/users", nil)
		failListResp, err := newUserClient.Do(reqFailList)
		Expect(err).NotTo(HaveOccurred())
		Expect(failListResp.StatusCode).To(Equal(http.StatusForbidden), "Normal user should not access admin list")
		failListResp.Body.Close()
	})

	It("Should get a single user by ID with predictions", func() {
		// 1. Admin creates a new user
		createReqBody := dto.CreateAdminRequest{
			Login:    "detailuser",
			Password: "detail_secure_123",
			Name:     "Detail User",
			Role:     models.RoleUser,
		}
		createBytes, _ := json.Marshal(createReqBody)
		reqCreate, _ := http.NewRequest(http.MethodPost, baseURL+"/admin/users", bytes.NewReader(createBytes))
		reqCreate.Header.Set("Content-Type", "application/json")

		createResp, err := client.Do(reqCreate)
		Expect(err).NotTo(HaveOccurred())

		createBody, _ := io.ReadAll(createResp.Body)
		Expect(createResp.StatusCode).To(Equal(http.StatusCreated), "Admin should create user. Body: %s", string(createBody))
		createResp.Body.Close()

		var createdUser dto.UserResponse
		err = json.Unmarshal(createBody, &createdUser)
		Expect(err).NotTo(HaveOccurred())
		Expect(createdUser.ID).NotTo(Equal(uuid.Nil))

		// 2. Admin gets user by ID
		reqGetUser, _ := http.NewRequest(http.MethodGet, baseURL+"/admin/users/"+createdUser.ID.String(), nil)
		getUserResp, err := client.Do(reqGetUser)
		Expect(err).NotTo(HaveOccurred())

		getUserBody, _ := io.ReadAll(getUserResp.Body)
		Expect(getUserResp.StatusCode).To(Equal(http.StatusOK), "Admin should get user by ID. Body: %s", string(getUserBody))
		getUserResp.Body.Close()

		var userDetail dto.AdminUserDetailResponse
		err = json.Unmarshal(getUserBody, &userDetail)
		Expect(err).NotTo(HaveOccurred())

		Expect(userDetail.ID).To(Equal(createdUser.ID))
		Expect(userDetail.Login).To(Equal("detailuser"))
		Expect(userDetail.Name).To(Equal("Detail User"))
		Expect(userDetail.Role).To(Equal(models.RoleUser))
		Expect(userDetail.Predictions).To(BeEmpty(), "New user should have no predictions")
		Expect(userDetail.Limit).To(Equal(100))
		Expect(userDetail.Offset).To(Equal(0))

		// 3. Admin gets user by ID with pagination params
		reqGetUserPaged, _ := http.NewRequest(http.MethodGet, baseURL+"/admin/users/"+createdUser.ID.String()+"?limit=5&offset=0", nil)
		getPagedResp, err := client.Do(reqGetUserPaged)
		Expect(err).NotTo(HaveOccurred())

		pagedBody, _ := io.ReadAll(getPagedResp.Body)
		Expect(getPagedResp.StatusCode).To(Equal(http.StatusOK), "Admin should get user with pagination. Body: %s", string(pagedBody))
		getPagedResp.Body.Close()

		var pagedDetail dto.AdminUserDetailResponse
		err = json.Unmarshal(pagedBody, &pagedDetail)
		Expect(err).NotTo(HaveOccurred())
		Expect(pagedDetail.Limit).To(Equal(5))
		Expect(pagedDetail.Offset).To(Equal(0))

		// 4. Invalid UUID returns 400
		reqInvalid, _ := http.NewRequest(http.MethodGet, baseURL+"/admin/users/not-a-uuid", nil)
		invalidResp, err := client.Do(reqInvalid)
		Expect(err).NotTo(HaveOccurred())
		Expect(invalidResp.StatusCode).To(Equal(http.StatusBadRequest), "Invalid UUID should return 400")
		invalidResp.Body.Close()

		// 5. Non-existent user returns 404
		reqNotFound, _ := http.NewRequest(http.MethodGet, baseURL+"/admin/users/"+uuid.New().String(), nil)
		notFoundResp, err := client.Do(reqNotFound)
		Expect(err).NotTo(HaveOccurred())
		Expect(notFoundResp.StatusCode).To(Equal(http.StatusNotFound), "Non-existent user should return 404")
		notFoundResp.Body.Close()

		// 6. Normal user cannot access this endpoint (RBAC)
		jar, _ := cookiejar.New(nil)
		normalClient := &http.Client{Jar: jar}
		normalLogin := dto.AuthRequest{
			Login:    "detailuser",
			Password: "detail_secure_123",
		}
		normalBytes, _ := json.Marshal(normalLogin)
		normalLoginResp, _ := normalClient.Post(baseURL+"/login", "application/json", bytes.NewReader(normalBytes))
		normalLoginResp.Body.Close()

		reqForbidden, _ := http.NewRequest(http.MethodGet, baseURL+"/admin/users/"+createdUser.ID.String(), nil)
		forbiddenResp, err := normalClient.Do(reqForbidden)
		Expect(err).NotTo(HaveOccurred())
		Expect(forbiddenResp.StatusCode).To(Equal(http.StatusForbidden), "Normal user should not access admin get user")
		forbiddenResp.Body.Close()
	})
})
