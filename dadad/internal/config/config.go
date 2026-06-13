package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config is the top-level application configuration loaded from configs/config.yaml and
// overridden by environment variables.
type Config struct {
	App    AppConfig    `mapstructure:"app"`
	Crypto CryptoConfig `mapstructure:"crypto"`

	// crank:config-fields
	Logging LoggingConfig `mapstructure:"logging"`
}

// AppConfig holds settings for the HTTP server itself.
type AppConfig struct {
	Name string `mapstructure:"name"`
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Env  string `mapstructure:"env"`
}

// CryptoConfig holds the encryption secret used by the crypto package.
// The secret is read from the CRYPTO_SECRET environment variable.
// Generate a strong secret with: openssl rand -base64 32
//
// IMPORTANT: Never commit the real secret. Use .env for local development
// and your platform's secret manager in production.
type CryptoConfig struct {
	Secret string `mapstructure:"secret"`
}

// crank:config-structs
// LoggingConfig controls slog output.
type LoggingConfig struct {
	Level     string `mapstructure:"level"`
	Format    string `mapstructure:"format"`
	AddSource bool   `mapstructure:"add_source"`
}

// Load reads configuration from configs/config.yaml and overrides values with environment
// variables (e.g. APP_PORT, DATABASE_PASSWORD, JWT_SECRET).
func Load() *Config {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("warning: could not read configs/config.yaml: %v\n", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("fatal: cannot parse configuration: %w", err))
	}
	return &cfg
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "dadad")
	v.SetDefault("app.host", "0.0.0.0")
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.env", "development")
	v.SetDefault("crypto.secret", "change-me-in-production")

	// crank:config-defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.add_source", false)
}
