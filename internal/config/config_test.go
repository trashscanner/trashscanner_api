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
	os.Setenv("AUTH_MANAGER_PUBLIC_KEY", "public-key")
	os.Setenv("AUTH_MANAGER_SECRET_PRIVATE_KEY", "private-key")
	defer func() {
		os.Unsetenv("CONFIG_PATH")
		os.Unsetenv("CONFIG_NAME")
		os.Unsetenv("AUTH_MANAGER_PUBLIC_KEY")
		os.Unsetenv("AUTH_MANAGER_SECRET_PRIVATE_KEY")
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
				AccessTokenTTL:   time.Minute * 15,
				RefreshTokenTTL:  time.Hour * 168,
				Algorithm:        "EdDSA",
				PublicKey:        "public-key",
				SecretPrivateKey: "private-key",
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
	})
}
