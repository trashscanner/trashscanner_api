package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gorilla/mux"
	authmocks "github.com/trashscanner/trashscanner_api/internal/auth/mocks"
	storemocks "github.com/trashscanner/trashscanner_api/internal/store/mocks"
)

func newTestServer(t *testing.T) (*Server, *storemocks.Store, *authmocks.AuthManager) {
	t.Helper()

	store := storemocks.NewStore(t)
	authManager := authmocks.NewAuthManager(t)

	srv := &Server{
		s:           &http.Server{},
		router:      mux.NewRouter(),
		store:       store,
		authManager: authManager,
	}

	return srv, store, authManager
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
