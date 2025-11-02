package dto

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

type HTTPResource interface {
	LoginUserRequest | ChangePasswordRequest
}

func GetRequestBody[T any](r *http.Request) (*T, error) {
	var body T
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}
	err := validator.New().Struct(body)
	if err != nil {
		return nil, err
	}

	return &body, nil
}

const (
	avatarFieldName = "avatar"
	scanFieldName   = "scan"
	maxFileSize     = 10 << 20 // 10 MB
)

var supportedFileTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
}

func GetAvatarFromMultipartForm(r *http.Request) (*models.File, error) {
	return getFileFromMultipartForm(r, avatarFieldName)
}

func GetScanFromMultipartForm(r *http.Request) (*models.File, error) {
	return getFileFromMultipartForm(r, scanFieldName)
}

func getFileFromMultipartForm(r *http.Request, fieldName string) (*models.File, error) {
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		return nil, errors.New("failed to parse multipart form")
	}

	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return nil, errors.New(fieldName + " field is required")
	}
	if header == nil {
		return nil, errors.New("file header is nil")
	}

	contentType := header.Header.Get("Content-Type")
	if _, ok := supportedFileTypes[contentType]; !ok {
		return nil, errors.New("unsupported file type: only image/jpeg and image/png are allowed")
	}

	if header.Size > maxFileSize {
		return nil, errors.New("file size exceeds the limit of 10MB")
	}

	if header.Size == 0 {
		return nil, errors.New("file is empty")
	}

	return &models.File{
		Name:  header.Filename,
		Size:  header.Size,
		Entry: file,
	}, nil
}
