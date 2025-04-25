package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Environment      string
	IsProduction     bool
	IsDevelopment    bool
	
	// Discord Bot Configuration
	DiscordToken     string
	DiscordGuild     string
	CommandPrefix    string
	
	// MongoDB Configuration
	MongoDBURI       string
	MongoDBURIWebcrawler string
	
	// Discord Channels
	ProductChannelID string
	
	// Crawler Configuration
	CrawlIntervalMinutes int
}

// Load loads the configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()
	
	cfg := &Config{
		Environment:     getEnv("ENVIRONMENT", "development"),
		DiscordToken:    getEnv("DISCORD_TOKEN", ""),
		DiscordGuild:    getEnv("DISCORD_GUILD", ""),
		CommandPrefix:   getEnv("COMMAND_PREFIX", "!"),
		MongoDBURI:      getEnv("MONGODB_URI", "mongodb://localhost:27017/hots"),
		MongoDBURIWebcrawler: getEnv("MONGODB_URI_WEBCRAWLER", "mongodb://localhost:27017/webcrawler"),
		ProductChannelID: getEnv("PRODUCT_CHANNEL_ID", ""),
	}
	
	// Derived properties
	cfg.IsProduction = cfg.Environment == "production"
	cfg.IsDevelopment = !cfg.IsProduction
	
	// Parse numeric values
	var err error
	cfg.CrawlIntervalMinutes, err = strconv.Atoi(getEnv("CRAWL_INTERVAL_MINUTES", "30"))
	if err != nil {
		cfg.CrawlIntervalMinutes = 30
	}
	
	// Validate required configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	
	return cfg, nil
}

// Validate checks if all required configuration is present
func (c *Config) Validate() error {
	if c.DiscordToken == "" {
		return fmt.Errorf("DISCORD_TOKEN environment variable is required")
	}
	
	// Add more validation as needed
	
	return nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}