package config

import (
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

const (
	devConfigPath = "config/dev"
	defaultName   = "config"
)

type Config struct {
	DB   DBConfig          `mapstructure:"database"`
	Auth AuthManagerConfig `mapstructure:"auth_manager"`
}

type DBConfig struct {
	Host     string `mapstructure:"host" validate:"required"`
	Port     string `mapstructure:"port" validate:"required"`
	User     string `mapstructure:"user" validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
	Name     string `mapstructure:"name" validate:"required"`

	MigrationsPath string `mapstructure:"migrations_path" validate:"required"`
	SSLMode        string `mapstructure:"sslmode" validate:"required,oneof=disable require verify-ca verify-full"`
}

type AuthManagerConfig struct {
	AccessTokenTTL   time.Duration `mapstructure:"access_token_ttl" validate:"required,gt=0"`
	RefreshTokenTTL  time.Duration `mapstructure:"refresh_token_ttl" validate:"required,gt=0"`
	Algorithm        string        `mapstructure:"signing_algorithm" validate:"required,oneof=HS256 HS384 HS512 RS256 RS384 RS512 ES256 ES384 ES512 EdDSA"`
	SecretPrivateKey string        `mapstructure:"secret_private_key"`
	PublicKey        string        `mapstructure:"public_key"`
}

func NewConfig() (Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = devConfigPath
	}
	name := os.Getenv("CONFIG_NAME")
	if name == "" {
		name = defaultName
	}

	v := viper.New()
	v.AddConfigPath(configPath)
	v.SetConfigName(name)
	v.SetConfigType("yaml")

	v.AutomaticEnv()
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var config Config
	if err := v.ReadInConfig(); err != nil {
		return config, err
	}
	if err := v.Unmarshal(&config); err != nil {
		return config, err
	}

	return config, validator.New().Struct(config)
}
