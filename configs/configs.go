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
		Port     string
		Env      string
		LogLevel string
	}
	Database struct {
		Host     string
		Port     string
		User     string
		Password string
		Name     string
		SSLMode  string
	}
	WebSocket struct {
		PingInterval   string
		MaxMessageSize int
	}
	Auth struct {
		SecretKey string
	}
	Features struct {
		EnableLogging    bool
		AllowCrossOrigin bool
	}
}

func LoadConfig() (*Config, error) {
	// Load .env file
	if err := godotenv.Load("./configs/.env"); err != nil {
		log.Info("No .env file found")
	}

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
