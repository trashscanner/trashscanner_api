package dto

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAvatarFromMultipartForm(t *testing.T) {
	t.Run("successfully uploads valid JPEG avatar", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreatePart(map[string][]string{
			"Content-Disposition": {`form-data; name="avatar"; filename="test_avatar.jpg"`},
			"Content-Type":        {"image/jpeg"},
		})
		require.NoError(t, err)

		imageData := []byte("fake jpeg image data")
		_, err = part.Write(imageData)
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		file, err := GetAvatarFromMultipartForm(req)

		require.NoError(t, err)
		require.NotNil(t, file)
		assert.Equal(t, "test_avatar.jpg", file.Name)
		assert.Equal(t, int64(len(imageData)), file.Size)
		assert.NotNil(t, file.Entry)

		content, err := io.ReadAll(file.Entry)
		require.NoError(t, err)
		assert.Equal(t, imageData, content)
	})

	t.Run("successfully uploads valid PNG avatar", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreatePart(map[string][]string{
			"Content-Disposition": {`form-data; name="avatar"; filename="test_avatar.png"`},
			"Content-Type":        {"image/png"},
		})
		require.NoError(t, err)

		imageData := []byte("fake png image data")
		_, err = part.Write(imageData)
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		file, err := GetAvatarFromMultipartForm(req)

		require.NoError(t, err)
		require.NotNil(t, file)
		assert.Equal(t, "test_avatar.png", file.Name)
	})

	t.Run("fails with missing avatar field", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("wrong_field", "test.jpg")
		require.NoError(t, err)
		_, err = part.Write([]byte("data"))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		file, err := GetAvatarFromMultipartForm(req)

		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "avatar field is required")
	})

	t.Run("fails with unsupported file type PDF", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreatePart(map[string][]string{
			"Content-Disposition": {`form-data; name="avatar"; filename="test.pdf"`},
			"Content-Type":        {"application/pdf"},
		})
		require.NoError(t, err)

		_, err = part.Write([]byte("fake pdf data"))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		file, err := GetAvatarFromMultipartForm(req)

		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "unsupported file type")
	})

	t.Run("fails with unsupported file type GIF", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreatePart(map[string][]string{
			"Content-Disposition": {`form-data; name="avatar"; filename="test.gif"`},
			"Content-Type":        {"image/gif"},
		})
		require.NoError(t, err)

		_, err = part.Write([]byte("fake gif data"))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		file, err := GetAvatarFromMultipartForm(req)

		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "unsupported file type")
	})

	t.Run("fails with file size exceeding limit", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreatePart(map[string][]string{
			"Content-Disposition": {`form-data; name="avatar"; filename="huge_avatar.jpg"`},
			"Content-Type":        {"image/jpeg"},
		})
		require.NoError(t, err)

		largeData := make([]byte, 11*1024*1024)
		_, err = part.Write(largeData)
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		file, err := GetAvatarFromMultipartForm(req)

		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "file size exceeds the limit")
	})

	t.Run("fails with empty file", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreatePart(map[string][]string{
			"Content-Disposition": {`form-data; name="avatar"; filename="empty.jpg"`},
			"Content-Type":        {"image/jpeg"},
		})
		require.NoError(t, err)

		_, err = part.Write([]byte{})
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		file, err := GetAvatarFromMultipartForm(req)

		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "file is empty")
	})

	t.Run("fails with invalid multipart form", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("invalid data"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=invalid")

		file, err := GetAvatarFromMultipartForm(req)

		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "failed to parse multipart form")
	})

	t.Run("fails with missing Content-Type header", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("avatar", "test.jpg")
		require.NoError(t, err)
		_, err = part.Write([]byte("data"))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/upload", body)

		file, err := GetAvatarFromMultipartForm(req)

		assert.Error(t, err)
		assert.Nil(t, file)
	})
}
