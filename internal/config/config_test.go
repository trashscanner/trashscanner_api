package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	os.Setenv("CONFIG_PATH", ".")
	os.Setenv("CONFIG_NAME", "test_config")
	defer func() {
		os.Unsetenv("CONFIG_PATH")
		os.Unsetenv("CONFIG_NAME")
	}()

	t.Run("successful config loading", func(t *testing.T) {
		config, err := NewConfig()
		expectedConfig := Config{
			DB: DBConfig{
				Host:           "localhost",
				Port:           "5432",
				User:           "testuser",
				Password:       "testpassword",
				Name:           "testdb",
				MigrationsPath: "file://internal/database/migrations",
				SSLMode:        "disable",
			},
			Auth: AuthManagerConfig{
				AccessTokenTTL:  time.Minute * 15,
				RefreshTokenTTL: time.Hour * 168,
				Algorithm:       "EdDSA",
			},
			Server: ServerConfig{
				Host: "0.0.0.0",
				Port: "8080",
			},
			Store: FileStoreConfig{
				Endpoint:  "localhost:9000",
				AccessKey: "minioadmin",
				SecretKey: "minioadmin",
				Bucket:    "trashscanner-images",
				UseSSL:    false,
			},
			Predictor: PredictorConfig{
				Address:                    "http://10.10.10.10:8000",
				Token:                      "token",
				MaxPredictionsInProcessing: 10,
			},
			AuthConfig: AuthConfig{
				DefaultRole: "anonymous",
			},
		}
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig, config)
	})

	t.Run("missing config file", func(t *testing.T) {
		oldEnv := os.Getenv("CONFIG_PATH")
		os.Setenv("CONFIG_PATH", "./nonexistent")
		defer func() {
			os.Setenv("CONFIG_PATH", oldEnv)
		}()
		_, err := NewConfig()
		assert.Error(t, err)
	})
	t.Run("config from env variable", func(t *testing.T) {
		oldEnvs := os.Environ()
		os.Setenv("DATABASE_HOST", "envhost")
		os.Setenv("DATABASE_PORT", "5433")
		os.Setenv("DATABASE_USER", "envuser")
		os.Setenv("DATABASE_PASSWORD", "envpassword")
		os.Setenv("DATABASE_NAME", "envdb")
		os.Setenv("FILESTORE_ENDPOINT", "minio.example.com:9000")
		os.Setenv("FILESTORE_ACCESS_KEY", "envAccessKey")
		os.Setenv("FILESTORE_SECRET_KEY", "envSecretKey")
		os.Setenv("FILESTORE_BUCKET", "env-bucket")
		os.Setenv("FILESTORE_USE_SSL", "true")
		os.Setenv("PREDICTOR_ADDRESS", "http://predictor.example.com:8000")
		os.Setenv("PREDICTOR_TOKEN", "env-token")
		os.Setenv("PREDICTOR_MAX_PREDICTIONS_IN_PROCESSING", "5")

		defer func() {
			for _, curEnv := range os.Environ() {
				parts := strings.SplitN(curEnv, "=", 2)
				os.Unsetenv(parts[0])
			}
			for _, e := range oldEnvs {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) == 2 {
					os.Setenv(parts[0], parts[1])
				} else {
					os.Unsetenv(parts[0])
				}
			}
		}()

		config, err := NewConfig()
		assert.NoError(t, err)
		assert.Equal(t, "envhost", config.DB.Host)
		assert.Equal(t, "5433", config.DB.Port)
		assert.Equal(t, "envuser", config.DB.User)
		assert.Equal(t, "envpassword", config.DB.Password)
		assert.Equal(t, "envdb", config.DB.Name)
		assert.Equal(t, "minio.example.com:9000", config.Store.Endpoint)
		assert.Equal(t, "envAccessKey", config.Store.AccessKey)
		assert.Equal(t, "envSecretKey", config.Store.SecretKey)
		assert.Equal(t, "env-bucket", config.Store.Bucket)
		assert.Equal(t, true, config.Store.UseSSL)
		assert.Equal(t, "http://predictor.example.com:8000", config.Predictor.Address)
		assert.Equal(t, "env-token", config.Predictor.Token)
		assert.Equal(t, 5, config.Predictor.MaxPredictionsInProcessing)
	})
}
