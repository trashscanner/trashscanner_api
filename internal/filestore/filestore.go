package filestore

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

const (
	fileStoreInitBucketTimeout = 10 * time.Second
	fileUploadTimeout          = 10 * time.Second
	avatarPathTmpl             = "%s/avatars/%s"
	scansPathTmpl              = "%s/scans/%s"
)

type FileStore interface {
	UpdateAvatar(ctx context.Context, user *models.User, file *models.File) error
	DeleteAvatar(ctx context.Context, avatarKey string) error
	UploadScan(ctx context.Context, userID string, file *models.File) (string, error)
}

type minioStore struct {
	client *minio.Client
	bucket string
}

func NewMinioStore(cfg config.Config) (FileStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fileStoreInitBucketTimeout)
	defer cancel()

	client, err := minio.New(cfg.Store.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Store.AccessKey, cfg.Store.SecretKey, ""),
		Secure: cfg.Store.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	store := &minioStore{client: client, bucket: cfg.Store.Bucket}
	err = client.MakeBucket(ctx, cfg.Store.Bucket, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := client.BucketExists(ctx, cfg.Store.Bucket)
		if errBucketExists == nil && exists {
			return store, nil
		}

		return nil, err
	}

	return store, nil
}

func (m *minioStore) UpdateAvatar(ctx context.Context, user *models.User, newAvatar *models.File) error {
	ctx, cancel := context.WithTimeout(ctx, fileUploadTimeout)
	defer cancel()

	if user.Avatar != nil {
		err := m.client.RemoveObject(ctx, m.bucket, *user.Avatar, minio.RemoveObjectOptions{ForceDelete: true})
		if err != nil {
			return err
		}
	}

	uploadInfo, err := m.client.PutObject(
		ctx, m.bucket,
		fmt.Sprintf(avatarPathTmpl, user.ID, newAvatar.Name),
		newAvatar.Entry, newAvatar.Size, minio.PutObjectOptions{},
	)
	if err != nil {
		return err
	}

	user.Avatar = &uploadInfo.Key

	return nil
}

func (m *minioStore) DeleteAvatar(ctx context.Context, avatarKey string) error {
	ctx, cancel := context.WithTimeout(ctx, fileUploadTimeout)
	defer cancel()

	return m.client.RemoveObject(ctx, m.bucket, avatarKey, minio.RemoveObjectOptions{ForceDelete: true})
}

func (m *minioStore) UploadScan(ctx context.Context, userID string, file *models.File) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, fileUploadTimeout)
	defer cancel()

	uploadInfo, err := m.client.PutObject(
		ctx, m.bucket,
		fmt.Sprintf(scansPathTmpl, userID, file.Name),
		file.Entry, file.Size, minio.PutObjectOptions{},
	)
	if err != nil {
		return "", err
	}

	return uploadInfo.Key, nil
}
