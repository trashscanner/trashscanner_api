package integration_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	miniotc "github.com/testcontainers/testcontainers-go/modules/minio"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/ssh"

	"github.com/trashscanner/trashscanner_api/internal/api"
	"github.com/trashscanner/trashscanner_api/internal/auth"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/filestore"
	"github.com/trashscanner/trashscanner_api/internal/logging"
	"github.com/trashscanner/trashscanner_api/internal/store"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var (
	ctx            context.Context
	cancel         context.CancelFunc
	pgContainer    *postgres.PostgresContainer
	minioContainer *miniotc.MinioContainer
	tsServer       *httptest.Server
	dbPool         *pgxpool.Pool

	// App instances
	pgStore     store.Store
	minioStore  filestore.FileStore
	authManager auth.AuthManager
	appConfig   config.Config
	apiServer   *api.Server
)

const (
	minioUser     = "minioadmin"
	minioPassword = "minioadmin"
	minioBucket   = "trashscanner-images"
)

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())

	// 1. Start PostgreSQL Container
	var err error
	pgContainer, err = postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("trashscanner_db"),
		postgres.WithUsername("trashscanner"),
		postgres.WithPassword("trashscanner"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(10*time.Second)),
	)
	Expect(err).NotTo(HaveOccurred(), "failed to start postgres container")

	dbURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	Expect(err).NotTo(HaveOccurred())

	// Run Migrations
	migrationsPath, err := filepath.Abs("../../internal/database/migrations")
	Expect(err).NotTo(HaveOccurred())
	m, err := migrate.New("file://"+migrationsPath, dbURL)
	Expect(err).NotTo(HaveOccurred(), "failed to init migrate")
	err = m.Up()
	if err != migrate.ErrNoChange {
		Expect(err).NotTo(HaveOccurred(), "failed to run migrations")
	}

	// 2. Start MinIO Container
	minioContainer, err = miniotc.Run(ctx,
		"minio/minio:latest",
		miniotc.WithUsername(minioUser),
		miniotc.WithPassword(minioPassword),
	)
	Expect(err).NotTo(HaveOccurred(), "failed to start minio container")

	minioHost, err := minioContainer.Endpoint(ctx, "")
	Expect(err).NotTo(HaveOccurred())

	// Initialize MinIO client & create bucket
	minioClient, err := minio.New(minioHost, &minio.Options{
		Creds:  credentials.NewStaticV4(minioUser, minioPassword, ""),
		Secure: false,
	})
	Expect(err).NotTo(HaveOccurred())

	err = minioClient.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{})
	Expect(err).NotTo(HaveOccurred())

	// 3. Initialize App Configuration & Layers
	os.Setenv("CONFIG_PATH", "./")
	os.Setenv("CONFIG_NAME", "test_config")

	appConfig, err = config.NewConfig()
	Expect(err).NotTo(HaveOccurred(), "failed to load config from test_config.yaml")

	// Override container-specific parameters
	appConfig.Store.Endpoint = minioHost
	appConfig.Store.AccessKey = minioUser
	appConfig.Store.SecretKey = minioPassword
	appConfig.Store.Bucket = minioBucket
	appConfig.Store.UseSSL = false

	// DB Pool
	dbConfig, err := pgxpool.ParseConfig(dbURL)
	Expect(err).NotTo(HaveOccurred())
	dbPool, err = pgxpool.NewWithConfig(ctx, dbConfig)
	Expect(err).NotTo(HaveOccurred())

	// Stores
	pgStore = store.NewPgStore(db.New(dbPool), func(tx db.DBTX) db.Querier { return db.New(tx) }, dbPool)
	minioStore, err = filestore.NewMinioStore(appConfig)
	Expect(err).NotTo(HaveOccurred())

	// Generate temporary EdDSA keys for test
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	Expect(err).NotTo(HaveOccurred())

	// Convert to PEM & base64 to mock environment variables
	privKeyBlock, err := ssh.MarshalPrivateKey(privKey, "")
	Expect(err).NotTo(HaveOccurred())
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	Expect(err).NotTo(HaveOccurred())

	os.Setenv("AUTH_MANAGER_SECRET_PRIVATE_KEY", base64.StdEncoding.EncodeToString(pem.EncodeToMemory(privKeyBlock)))
	os.Setenv("AUTH_MANAGER_PUBLIC_KEY", base64.StdEncoding.EncodeToString(ssh.MarshalAuthorizedKey(sshPubKey)))

	// AuthManager init
	authManager, err = auth.NewJWTManager(appConfig, pgStore)
	Expect(err).NotTo(HaveOccurred())

	logger := logging.NewLogger(config.Config{Log: config.LogConfig{Level: "info"}})

	// API Server
	apiServer = api.NewServer(appConfig, pgStore, minioStore, authManager, nil, logger)
	tsServer = httptest.NewServer(apiServer.InitRouter())

	fmt.Printf("Test server listening on: %s\n", tsServer.URL)
})

var _ = AfterSuite(func() {
	if tsServer != nil {
		tsServer.Close()
	}
	if dbPool != nil {
		dbPool.Close()
	}
	if pgContainer != nil {
		err := pgContainer.Terminate(ctx)
		Expect(err).NotTo(HaveOccurred())
	}
	if minioContainer != nil {
		err := minioContainer.Terminate(ctx)
		Expect(err).NotTo(HaveOccurred())
	}
	cancel()
})
