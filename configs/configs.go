package configs

import (
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port     string `mapstructure:"port"`
		Env      string `mapstructure:"env"`
		LogLevel string `mapstructure:"log_level"`
	} `mapstructure:"server"`
	Database struct {
		DatabaseUrl string `mapstructure:"database_url"`
	} `mapstructure:"database"`
	WebSocket struct {
		PingInterval   string `mapstructure:"ping_interval"`
		MaxMessageSize int    `mapstructure:"max_message_size"`
	} `mapstructure:"websocket"`
	Auth struct {
		SecretKey string `mapstructure:"secret_key"`
	} `mapstructure:"auth"`
	Features struct {
		EnableLogging    bool `mapstructure:"enable_logging"`
		AllowCrossOrigin bool `mapstructure:"allow_cross_origin"`
	} `mapstructure:"features"`
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load("./configs/.env"); err != nil {
		log.Info("No .env file found")
	}

	// Set up viper
	viper.SetConfigName("config")    // Name of the config file (without extension)
	viper.SetConfigType("yaml")      // Config file type
	viper.AddConfigPath("./configs") // Path to look for the config file
	viper.AutomaticEnv()             // Automatically map environment variables

	// Allow dots in environment variables to map to nested keys
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	// Manually substitute environment variables in the config
	substituteEnvVarsInConfig()

	// Unmarshal the config into a struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Helper function to manually replace environment variables in config file values
func substituteEnvVarsInConfig() {
	// Iterate over each key-value pair in viper's config
	for _, key := range viper.AllKeys() {
		// Get the current value
		value := viper.GetString(key)

		// Check if the value contains environment variable syntax (e.g., ${PORT})
		if strings.Contains(value, "${") {
			// Replace environment variables in the value (e.g., ${PORT})
			replacedValue := os.Expand(value, func(name string) string {
				// Lookup the environment variable's value
				return os.Getenv(name)
			})

			// Set the replaced value back into viper
			viper.Set(key, replacedValue)

		}
	}
}
