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
	Server    ServerConfig      `mapstructure:"server"`
	DB        DBConfig          `mapstructure:"database"`
	Store     FileStoreConfig   `mapstructure:"filestore"`
	Auth      AuthManagerConfig `mapstructure:"auth_manager"`
	Predictor PredictorConfig   `mapstructure:"predictor"`
	Log       LogConfig         `mapstructure:"log"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
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

type FileStoreConfig struct {
	Endpoint  string `mapstructure:"endpoint" validate:"required"`
	AccessKey string `mapstructure:"access_key" validate:"required"`
	SecretKey string `mapstructure:"secret_key" validate:"required"`
	UseSSL    bool   `mapstructure:"use_ssl"`
	Bucket    string `mapstructure:"bucket" validate:"required"`
}

type AuthManagerConfig struct {
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl" validate:"required,gt=0"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl" validate:"required,gt=0"`
	Algorithm       string        `mapstructure:"signing_algorithm" validate:"required,oneof=EdDSA"`
}

type PredictorConfig struct {
	Host                       string `mapstructure:"host" validate:"required"`
	Token                      string `mapstructure:"token" validate:"required"`
	MaxPredictionsInProcessing int    `mapstructure:"max_predictions_in_processing" validate:"required,gt=0"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	File   string `mapstructure:"file"`
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
	setDefaults(v)

	var config Config
	if err := v.ReadInConfig(); err != nil {
		return config, err
	}
	if err := v.Unmarshal(&config); err != nil {
		return config, err
	}

	return config, validator.New().Struct(config)
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", "8080")
}
