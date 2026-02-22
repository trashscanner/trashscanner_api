package config

import (
	"os"
	"path"
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
	Server     ServerConfig      `mapstructure:"server"`
	DB         DBConfig          `mapstructure:"database"`
	Store      FileStoreConfig   `mapstructure:"filestore"`
	Auth       AuthManagerConfig `mapstructure:"auth_manager"`
	Predictor  PredictorConfig   `mapstructure:"predictor"`
	Log        LogConfig         `mapstructure:"log"`
	AuthConfig AuthConfig        `mapstructure:"auth_config"`
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
	Address                    string `mapstructure:"address" validate:"required"`
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
	v.SetDefault("auth_config.default_role", "anonymous")
}

type AuthRule struct {
	Pattern string   `mapstructure:"pattern" yaml:"pattern"`
	Roles   []string `mapstructure:"roles" yaml:"roles"`
}

type AuthConfig struct {
	Rules       []AuthRule `mapstructure:"rules" yaml:"rules"`
	DefaultRole string     `mapstructure:"default_role" yaml:"default_role"`
}

func (c *AuthConfig) IsAllowed(role, urlPath string) bool {
	urlPath = path.Clean(urlPath)

	for _, rule := range c.Rules {
		if matchPattern(rule.Pattern, urlPath) {
			for _, r := range rule.Roles {
				if r == role {
					return true
				}
			}
			return false
		}
	}

	return false
}

func matchPattern(pattern, urlPath string) bool {
	patternParts := splitPath(pattern)
	pathParts := splitPath(urlPath)

	return matchParts(patternParts, pathParts)
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return []string{}
	}
	return strings.Split(p, "/")
}

func matchParts(pattern, path []string) bool {
	pi, pa := 0, 0

	for pi < len(pattern) && pa < len(path) {
		if pattern[pi] == "**" {
			if pi == len(pattern)-1 {
				return true
			}
			for i := pa; i <= len(path); i++ {
				if matchParts(pattern[pi+1:], path[i:]) {
					return true
				}
			}
			return false
		}

		if pattern[pi] == "*" {
			pi++
			pa++
			continue
		}

		if pattern[pi] != path[pa] {
			return false
		}

		pi++
		pa++
	}

	for pi < len(pattern) && pattern[pi] == "**" {
		pi++
	}

	return pi == len(pattern) && pa == len(path)
}
