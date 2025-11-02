package filestore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

var (
	testConfig   config.Config
	minioClient  *minio.Client
	testBucket   = "test-bucket"
	testEndpoint = "localhost:9000"
)

func TestMain(m *testing.M) {
	testConfig = config.Config{
		Store: config.FileStoreConfig{
			Endpoint:  testEndpoint,
			AccessKey: "minioadmin",
			SecretKey: "minioadmin",
			UseSSL:    false,
			Bucket:    testBucket,
		},
	}

	var err error
	minioClient, err = minio.New(testEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("Failed to create MinIO client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := minioClient.BucketExists(ctx, testBucket)
	if err != nil {
		log.Fatalf("Failed to check bucket existence: %v", err)
	}

	if !exists {
		log.Printf("MinIO test bucket '%s' does not exist. Make sure MinIO is running on %s", testBucket, testEndpoint)
	}

	code := m.Run()

	cleanupAllBuckets()

	os.Exit(code)
}

func cleanupAllBuckets() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := minioClient.BucketExists(ctx, testBucket)
	if err == nil && exists {
		objectsCh := minioClient.ListObjects(ctx, testBucket, minio.ListObjectsOptions{
			Recursive: true,
		})
		for object := range objectsCh {
			if object.Err == nil {
				_ = minioClient.RemoveObject(ctx, testBucket, object.Key, minio.RemoveObjectOptions{})
			}
		}
		_ = minioClient.RemoveBucket(ctx, testBucket)
	}
}

func setupTestBucket(t *testing.T) func() {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := minioClient.BucketExists(ctx, testBucket)
	if err == nil && exists {
		objectsCh := minioClient.ListObjects(ctx, testBucket, minio.ListObjectsOptions{
			Recursive: true,
		})
		for object := range objectsCh {
			if object.Err == nil {
				_ = minioClient.RemoveObject(ctx, testBucket, object.Key, minio.RemoveObjectOptions{})
			}
		}
		_ = minioClient.RemoveBucket(ctx, testBucket)
	}

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		objectsCh := minioClient.ListObjects(ctx, testBucket, minio.ListObjectsOptions{
			Recursive: true,
		})
		for object := range objectsCh {
			if object.Err == nil {
				_ = minioClient.RemoveObject(ctx, testBucket, object.Key, minio.RemoveObjectOptions{})
			}
		}
		_ = minioClient.RemoveBucket(ctx, testBucket)
	}
}

func TestNewMinioStore(t *testing.T) {
	t.Run("successfully creates store and bucket", func(t *testing.T) {
		cleanup := setupTestBucket(t)
		defer cleanup()

		store, err := NewMinioStore(testConfig)

		require.NoError(t, err)
		require.NotNil(t, store)

		minioStore := store.(*minioStore)
		ctx := context.Background()
		exists, err := minioStore.client.BucketExists(ctx, testBucket)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("works when bucket already exists", func(t *testing.T) {
		cleanup := setupTestBucket(t)
		defer cleanup()

		store1, err := NewMinioStore(testConfig)
		require.NoError(t, err)
		require.NotNil(t, store1)

		store2, err := NewMinioStore(testConfig)
		require.NoError(t, err)
		require.NotNil(t, store2)
	})

	t.Run("fails with invalid credentials", func(t *testing.T) {
		cfg := testConfig
		cfg.Store.AccessKey = "invalid"
		cfg.Store.SecretKey = "invalid"

		store, err := NewMinioStore(cfg)
		assert.Error(t, err)
		assert.Nil(t, store)
	})
}

func TestUpdateAvatar(t *testing.T) {
	t.Run("successfully uploads new avatar", func(t *testing.T) {
		cleanup := setupTestBucket(t)
		defer cleanup()

		store, err := NewMinioStore(testConfig)
		require.NoError(t, err)

		userID := uuid.New()
		user := &models.User{
			ID:     userID,
			Avatar: nil,
		}

		avatarContent := []byte("fake avatar image content")
		file := models.File{
			Name:  "avatar.jpg",
			Size:  int64(len(avatarContent)),
			Entry: io.NopCloser(bytes.NewReader(avatarContent)),
		}

		ctx := context.Background()
		err = store.UpdateAvatar(ctx, user, &file)

		require.NoError(t, err)
		require.NotNil(t, user.Avatar)

		minioStore := store.(*minioStore)
		obj, err := minioStore.client.GetObject(ctx, testBucket, *user.Avatar, minio.GetObjectOptions{})
		require.NoError(t, err)
		defer obj.Close()

		downloadedContent, err := io.ReadAll(obj)
		require.NoError(t, err)
		assert.Equal(t, avatarContent, downloadedContent)
	})

	t.Run("replaces existing avatar", func(t *testing.T) {
		cleanup := setupTestBucket(t)
		defer cleanup()

		store, err := NewMinioStore(testConfig)
		require.NoError(t, err)

		userID := uuid.New()
		user := &models.User{
			ID:     userID,
			Avatar: nil,
		}

		oldAvatarContent := []byte("old avatar")
		oldFile := models.File{
			Name:  "old_avatar.jpg",
			Size:  int64(len(oldAvatarContent)),
			Entry: io.NopCloser(bytes.NewReader(oldAvatarContent)),
		}

		ctx := context.Background()
		err = store.UpdateAvatar(ctx, user, &oldFile)
		require.NoError(t, err)
		oldAvatarPath := *user.Avatar

		newAvatarContent := []byte("new avatar content")
		newFile := models.File{
			Name:  "new_avatar.png",
			Size:  int64(len(newAvatarContent)),
			Entry: io.NopCloser(bytes.NewReader(newAvatarContent)),
		}

		err = store.UpdateAvatar(ctx, user, &newFile)
		require.NoError(t, err)
		require.NotNil(t, user.Avatar)

		minioStore := store.(*minioStore)
		obj, err := minioStore.client.GetObject(ctx, testBucket, *user.Avatar, minio.GetObjectOptions{})
		require.NoError(t, err)
		defer obj.Close()

		downloadedContent, err := io.ReadAll(obj)
		require.NoError(t, err)
		assert.Equal(t, newAvatarContent, downloadedContent)

		_, err = minioStore.client.StatObject(ctx, testBucket, oldAvatarPath, minio.StatObjectOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "The specified key does not exist")
	})

	t.Run("handles upload timeout", func(t *testing.T) {
		cleanup := setupTestBucket(t)
		defer cleanup()

		store, err := NewMinioStore(testConfig)
		require.NoError(t, err)

		userID := uuid.New()
		user := &models.User{
			ID:     userID,
			Avatar: nil,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		avatarContent := []byte("avatar content")
		file := models.File{
			Name:  "avatar.jpg",
			Size:  int64(len(avatarContent)),
			Entry: io.NopCloser(bytes.NewReader(avatarContent)),
		}

		err = store.UpdateAvatar(ctx, user, &file)
		assert.Error(t, err)
	})
}

func TestUploadScan(t *testing.T) {
	t.Run("successfully uploads scan", func(t *testing.T) {
		cleanup := setupTestBucket(t)
		defer cleanup()

		store, err := NewMinioStore(testConfig)
		require.NoError(t, err)

		userID := uuid.New().String()
		scanID := uuid.New()
		scanContent := []byte("fake scan image content")
		file := models.File{
			Name:  "trash_photo.jpg",
			ID:    scanID,
			Size:  int64(len(scanContent)),
			Entry: io.NopCloser(bytes.NewReader(scanContent)),
		}

		ctx := context.Background()
		scanPath, err := store.UploadScan(ctx, userID, &file)

		require.NoError(t, err)
		assert.NotEmpty(t, scanPath)

		// Проверяем, что путь содержит ID файла, а не имя
		expectedPath := fmt.Sprintf("%s/scans/%s", userID, scanID.String())
		assert.Equal(t, expectedPath, scanPath)

		minioStore := store.(*minioStore)
		obj, err := minioStore.client.GetObject(ctx, testBucket, scanPath, minio.GetObjectOptions{})
		require.NoError(t, err)
		defer obj.Close()

		downloadedContent, err := io.ReadAll(obj)
		require.NoError(t, err)
		assert.Equal(t, scanContent, downloadedContent)
	})

	t.Run("uploads multiple scans for same user", func(t *testing.T) {
		cleanup := setupTestBucket(t)
		defer cleanup()

		store, err := NewMinioStore(testConfig)
		require.NoError(t, err)

		userID := uuid.New().String()
		ctx := context.Background()

		scanPaths := make([]string, 0, 3)
		for i := 0; i < 3; i++ {
			scanID := uuid.New()
			content := []byte(fmt.Sprintf("scan content %d", i))
			file := models.File{
				ID:    scanID,
				Name:  fmt.Sprintf("scan_%d.jpg", i),
				Size:  int64(len(content)),
				Entry: io.NopCloser(bytes.NewReader(content)),
			}

			scanPath, err := store.UploadScan(ctx, userID, &file)
			require.NoError(t, err)
			scanPaths = append(scanPaths, scanPath)

			// Проверяем, что путь содержит ID файла
			expectedPath := fmt.Sprintf("%s/scans/%s", userID, scanID.String())
			assert.Equal(t, expectedPath, scanPath)
		}

		assert.Len(t, scanPaths, 3)
		// Проверяем, что все пути уникальны
		for i := 0; i < 3; i++ {
			for j := i + 1; j < 3; j++ {
				assert.NotEqual(t, scanPaths[i], scanPaths[j])
			}
		}

		minioStore := store.(*minioStore)
		for i, scanPath := range scanPaths {
			obj, err := minioStore.client.GetObject(ctx, testBucket, scanPath, minio.GetObjectOptions{})
			require.NoError(t, err)

			content, err := io.ReadAll(obj)
			require.NoError(t, err)
			obj.Close()

			expectedContent := fmt.Sprintf("scan content %d", i)
			assert.Equal(t, expectedContent, string(content))
		}
	})

	t.Run("handles invalid reader", func(t *testing.T) {
		cleanup := setupTestBucket(t)
		defer cleanup()

		store, err := NewMinioStore(testConfig)
		require.NoError(t, err)

		userID := uuid.New().String()

		errorReader := &errorReader{err: fmt.Errorf("read error")}
		file := models.File{
			ID:    uuid.New(),
			Name:  "scan.jpg",
			Size:  100,
			Entry: errorReader,
		}

		ctx := context.Background()
		_, err = store.UploadScan(ctx, userID, &file)

		assert.Error(t, err)
	})
}

func TestFileStoreIntegration(t *testing.T) {
	t.Run("complete user flow with avatar and scans", func(t *testing.T) {
		cleanup := setupTestBucket(t)
		defer cleanup()

		store, err := NewMinioStore(testConfig)
		require.NoError(t, err)

		ctx := context.Background()
		userID := uuid.New()
		user := &models.User{
			ID:     userID,
			Avatar: nil,
		}

		avatarContent := []byte("user avatar image")
		avatarFile := models.File{
			Name:  "avatar.jpg",
			Size:  int64(len(avatarContent)),
			Entry: io.NopCloser(bytes.NewReader(avatarContent)),
		}

		err = store.UpdateAvatar(ctx, user, &avatarFile)
		require.NoError(t, err)
		assert.NotNil(t, user.Avatar)

		scanCount := 5
		scanPaths := make([]string, 0, scanCount)
		for i := 0; i < scanCount; i++ {
			scanID := uuid.New()
			content := []byte(fmt.Sprintf("trash scan %d", i))
			scanFile := models.File{
				ID:    scanID,
				Name:  fmt.Sprintf("trash_%d.jpg", i),
				Size:  int64(len(content)),
				Entry: io.NopCloser(bytes.NewReader(content)),
			}

			scanPath, err := store.UploadScan(ctx, userID.String(), &scanFile)
			require.NoError(t, err)
			scanPaths = append(scanPaths, scanPath)
		}

		newAvatarContent := []byte("new avatar image")
		newAvatarFile := models.File{
			Name:  "new_avatar.png",
			Size:  int64(len(newAvatarContent)),
			Entry: io.NopCloser(bytes.NewReader(newAvatarContent)),
		}

		err = store.UpdateAvatar(ctx, user, &newAvatarFile)
		require.NoError(t, err)

		minioStore := store.(*minioStore)
		for _, scanPath := range scanPaths {
			_, err := minioStore.client.StatObject(ctx, testBucket, scanPath, minio.StatObjectOptions{})
			assert.NoError(t, err, "scan should exist: %s", scanPath)
		}

		obj, err := minioStore.client.GetObject(ctx, testBucket, *user.Avatar, minio.GetObjectOptions{})
		require.NoError(t, err)
		defer obj.Close()

		downloadedAvatar, err := io.ReadAll(obj)
		require.NoError(t, err)
		assert.Equal(t, newAvatarContent, downloadedAvatar)
	})
}

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func (e *errorReader) Close() error {
	return nil
}

func BenchmarkUpdateAvatar(b *testing.B) {
	ctx := context.Background()
	_ = minioClient.MakeBucket(ctx, testBucket, minio.MakeBucketOptions{})

	store, err := NewMinioStore(testConfig)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}

	b.Cleanup(func() {
		objectsCh := minioClient.ListObjects(ctx, testBucket, minio.ListObjectsOptions{Recursive: true})
		for object := range objectsCh {
			if object.Err == nil {
				_ = minioClient.RemoveObject(ctx, testBucket, object.Key, minio.RemoveObjectOptions{})
			}
		}
		_ = minioClient.RemoveBucket(ctx, testBucket)
	})

	avatarContent := []byte(strings.Repeat("a", 1024*100))
	userID := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user := &models.User{
			ID:     userID,
			Avatar: nil,
		}

		file := models.File{
			Name:  fmt.Sprintf("avatar_%d.jpg", i),
			Size:  int64(len(avatarContent)),
			Entry: io.NopCloser(bytes.NewReader(avatarContent)),
		}

		err := store.UpdateAvatar(ctx, user, &file)
		if err != nil {
			b.Fatalf("failed to upload avatar: %v", err)
		}
	}
}

func BenchmarkUploadScan(b *testing.B) {
	ctx := context.Background()
	_ = minioClient.MakeBucket(ctx, testBucket, minio.MakeBucketOptions{})

	store, err := NewMinioStore(testConfig)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}

	b.Cleanup(func() {
		objectsCh := minioClient.ListObjects(ctx, testBucket, minio.ListObjectsOptions{Recursive: true})
		for object := range objectsCh {
			if object.Err == nil {
				_ = minioClient.RemoveObject(ctx, testBucket, object.Key, minio.RemoveObjectOptions{})
			}
		}
		_ = minioClient.RemoveBucket(ctx, testBucket)
	})

	scanContent := []byte(strings.Repeat("s", 1024*200))
	userID := uuid.New().String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file := models.File{
			ID:    uuid.New(),
			Name:  fmt.Sprintf("scan_%d.jpg", i),
			Size:  int64(len(scanContent)),
			Entry: io.NopCloser(bytes.NewReader(scanContent)),
		}

		_, err := store.UploadScan(ctx, userID, &file)
		if err != nil {
			b.Fatalf("failed to upload scan: %v", err)
		}
	}
}
