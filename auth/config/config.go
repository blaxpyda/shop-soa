package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App        AppConfig        `mapstructure:"app"`
	SQLLite    SQLLiteConfig    `mapstructure:"sqlite"`
	GRPC       GRPCConfig       `mapstructure:"grpc"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	SMS        SMSConfig        `mapstructure:"easysend_sms"`
	Gmail      GmailConfig      `mapstructure:"gmail"`
	SuperAdmin SuperAdminConfig `mapstructure:"super_admin"`
}

type AppConfig struct {
	Name         string `mapstructure:"name"`
	Environment  string `mapstructure:"environment"`
	Port         int    `mapstructure:"port"`
	LogLevel     string `mapstructure:"log_level"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

type SMSConfig struct {
	APIKey   string `mapstructure:"api_key"`
	SenderID string `mapstructure:"sender_id"`
}

type GmailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type SQLLiteConfig struct {
	Path string `mapstructure:"path"`
}

type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

type JWTConfig struct {
	PrivateKeyPath string `mapstructure:"private_key_path"`
	PublicKeyPath  string `mapstructure:"public_key_path"`
	AccessTokenTTL int    `mapstructure:"access_token_ttl"`
}

type SuperAdminConfig struct {
	Email     string `mapstructure:"email"`
	Password  string `mapstructure:"password"`
	FirstName string `mapstructure:"first_name"`
	LastName  string `mapstructure:"last_name"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	_ = godotenv.Load()

	viper.SetEnvPrefix("AUTH")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.SQLLite.Path == "" {
		return nil, fmt.Errorf("sqlite.path is required")
	}

	return &cfg, nil
}
