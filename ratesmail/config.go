package ratesmail

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port        int
	BindAddress string `mapstructure:"bind_address"`
}

type EmailConfig struct {
	Username string
	Password string
	From     string
	SMTP     struct {
		Host string
		Port int
		SSL  bool
	}
}

type DBConfig struct {
	Filename string
	WALMode  bool `mapstructure:"wal_mode"`
}

type ExchangeRateConfig struct {
	From string
	To   string
}

type ApplicationConfig struct {
	DB            DBConfig
	Server        ServerConfig
	Email         EmailConfig
	ExchangeRates ExchangeRateConfig `mapstructure:"exchange_rates"`
}

func ParseConfig() (ApplicationConfig, error) {
	mode := os.Getenv("APP_MODE")
	if mode == "" {
		mode = "dev"
	}

	// Load config in order:
	// - config.*
	// - config.(dev|prod).*
	// - config.local.*
	// - Environment variables
	// * refers to (json|toml|yaml|yml)

	viper.SetConfigName("config")
	viper.AddConfigPath("./conf")
	viper.ReadInConfig()

	viper.SetConfigName("config." + mode)
	viper.MergeInConfig()

	viper.SetConfigName("config.local")
	viper.MergeInConfig()

	viper.AutomaticEnv()
	viper.BindEnv("email.username", "EMAIL_USERNAME")
	viper.BindEnv("email.password", "EMAIL_PASSWORD")
	viper.BindEnv("email.from", "EMAIL_FROM")
	viper.BindEnv("email.smtp.host", "EMAIL_SMTP_HOST")
	viper.MergeInConfig()

	var config *ApplicationConfig
	err := viper.Unmarshal(&config)
	if err != nil {
		return *config, fmt.Errorf("Failed to parse config file: %s", err)
	}

	return *config, nil
}
