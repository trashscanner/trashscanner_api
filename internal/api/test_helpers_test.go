package api

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	authmocks "github.com/trashscanner/trashscanner_api/internal/auth/mocks"
	"github.com/trashscanner/trashscanner_api/internal/config"
	filestoremocks "github.com/trashscanner/trashscanner_api/internal/filestore/mocks"
	"github.com/trashscanner/trashscanner_api/internal/logging"
	storemocks "github.com/trashscanner/trashscanner_api/internal/store/mocks"
)

func newTestServer(t *testing.T) (*Server, *storemocks.Store, *authmocks.AuthManager, *filestoremocks.FileStore, *mockPredictor) {
	t.Helper()

	store := storemocks.NewStore(t)
	authManager := authmocks.NewAuthManager(t)
	fileStore := filestoremocks.NewFileStore(t)
	predictor := newMockPredictor(t)
	cfg := config.Config{Log: config.LogConfig{Level: "error", Format: "text"}}
	logger := logging.NewLogger(cfg)

	srv := &Server{
		s:           &http.Server{},
		router:      mux.NewRouter(),
		store:       store,
		authManager: authManager,
		fileStore:   fileStore,
		predictor:   predictor,
		logger:      logger,
	}

	return srv, store, authManager, fileStore, predictor
}

// multipartFormData holds the created multipart form data
type multipartFormData struct {
	body        io.Reader
	contentType string
}

// createMultipartFormWithAvatar creates a multipart form with an avatar file
func createMultipartFormWithAvatar(t *testing.T, filename, contentType string, data []byte) multipartFormData {
	t.Helper()
	return createMultipartFormWithField(t, "avatar", filename, contentType, data)
}

// createMultipartFormWithField creates a multipart form with a custom field name
func createMultipartFormWithField(t *testing.T, fieldName, filename, contentType string, data []byte) multipartFormData {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="` + fieldName + `"; filename="` + filename + `"`},
		"Content-Type":        {contentType},
	})
	require.NoError(t, err)

	_, err = part.Write(data)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	return multipartFormData{
		body:        body,
		contentType: writer.FormDataContentType(),
	}
}

func loadJSONFixture(t testing.TB, name string) []byte {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine caller path")
	}

	baseDir := filepath.Dir(filename)
	path := filepath.Join(baseDir, "..", "testdata", "rest_data", name)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open fixture %s: %v", name, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			t.Fatalf("failed to close fixture %s: %v", name, cerr)
		}
	}()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}

	return data
}

func loadJSONFixtureReader(t testing.TB, name string) io.Reader {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine caller path")
	}

	baseDir := filepath.Dir(filename)
	path := filepath.Join(baseDir, "..", "testdata", "rest_data", name)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open fixture %s: %v", name, err)
	}

	return f
}
