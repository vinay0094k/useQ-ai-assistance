package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	AI       AIConfig       `mapstructure:"ai"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Vector   VectorConfig   `mapstructure:"vector"`
}

// AppConfig holds application settings
type AppConfig struct {
	Name        string   `mapstructure:"name"`
	Version     string   `mapstructure:"version"`
	ProjectRoot string   `mapstructure:"project_root"`
	Extensions  []string `mapstructure:"extensions"`
	ExcludeDirs []string `mapstructure:"exclude_dirs"`
}

// DatabaseConfig holds database settings
type DatabaseConfig struct {
	Path    string `mapstructure:"path"`
	Timeout string `mapstructure:"timeout"`
}

// AIConfig holds AI provider settings
type AIConfig struct {
	Primary   string            `mapstructure:"primary"`
	Fallbacks []string          `mapstructure:"fallbacks"`
	OpenAI    ProviderConfig    `mapstructure:"openai"`
	Gemini    ProviderConfig    `mapstructure:"gemini"`
}

// ProviderConfig holds provider-specific settings
type ProviderConfig struct {
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level     string `mapstructure:"level"`
	EnableLog bool   `mapstructure:"enable_log"`
	LogDir    string `mapstructure:"log_dir"`
}

// VectorConfig holds vector database settings
type VectorConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Collection string `mapstructure:"collection"`
	Dimension  int    `mapstructure:"dimension"`
}

// Load loads configuration from environment and files
func Load() (*Config, error) {
	// Set defaults
	viper.SetDefault("app.name", "useQ AI Assistant")
	viper.SetDefault("app.version", "1.0.0")
	viper.SetDefault("app.project_root", ".")
	viper.SetDefault("app.extensions", []string{".go", ".js", ".py", ".md"})
	viper.SetDefault("app.exclude_dirs", []string{"vendor", "node_modules", ".git"})
	
	viper.SetDefault("database.path", "storage/useq.db")
	viper.SetDefault("database.timeout", "30s")
	
	viper.SetDefault("ai.primary", "openai")
	viper.SetDefault("ai.fallbacks", []string{"gemini"})
	viper.SetDefault("ai.openai.model", "gpt-4-turbo-preview")
	viper.SetDefault("ai.openai.max_tokens", 4000)
	viper.SetDefault("ai.openai.temperature", 0.1)
	
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.enable_log", true)
	viper.SetDefault("logging.log_dir", "./logs")
	
	viper.SetDefault("vector.host", "localhost")
	viper.SetDefault("vector.port", 6333)
	viper.SetDefault("vector.collection", "code_embeddings")
	viper.SetDefault("vector.dimension", 1536)

	// Bind environment variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read from environment
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		viper.Set("ai.openai.api_key", apiKey)
	}
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		viper.Set("ai.gemini.api_key", apiKey)
	}

	// Try to read config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// GetTimeout parses timeout string to duration
func (c *Config) GetTimeout() time.Duration {
	if duration, err := time.ParseDuration(c.Database.Timeout); err == nil {
		return duration
	}
	return 30 * time.Second
}