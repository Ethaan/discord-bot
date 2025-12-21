package config

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Port             string
	DocsPort         string
	DiscordToken     string
	DiscordGuildID   string
	ParentCategoryID string
	TibiaAPIURL      string
	Database         DatabaseConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func Load() (*Config, error) {
	if err := loadEnvFile(".env"); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	var dbConfig DatabaseConfig

	if databaseURL := getEnv("DATABASE_URL", ""); databaseURL != "" {
		var err error
		dbConfig, err = parseDatabaseURL(databaseURL)
		if err != nil {
			return nil, fmt.Errorf("error parsing DATABASE_URL: %w", err)
		}
	} else {
		dbConfig = DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "bot_user"),
			Password: getEnv("DB_PASSWORD", "bot_password"),
			Name:     getEnv("DB_NAME", "discord_bot"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		}
	}

	cfg := &Config{
		Port:             getEnv("PORT", "8080"),
		DocsPort:         getEnv("DOCS_PORT", "8081"),
		DiscordToken:     getEnv("DISCORD_BOT_TOKEN", ""),
		DiscordGuildID:   getEnv("DISCORD_GUILD_ID", ""),
		ParentCategoryID: getEnv("PARENT_CATEGORY_ID", ""),
		TibiaAPIURL:      getEnv("TIBIA_API_URL", "https://api.tibiadata.com/v4"),
		Database:         dbConfig,
	}

	return cfg, nil
}

// parseDatabaseURL parses a PostgreSQL connection URL
// Format: postgresql://user:password@host:port/database?sslmode=require
func parseDatabaseURL(databaseURL string) (DatabaseConfig, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return DatabaseConfig{}, err
	}

	password, _ := u.User.Password()

	sslMode := u.Query().Get("sslmode")
	if sslMode == "" {
		sslMode = "require"
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}

	dbName := strings.TrimPrefix(u.Path, "/")

	return DatabaseConfig{
		Host:     host,
		Port:     port,
		User:     u.User.Username(),
		Password: password,
		Name:     dbName,
		SSLMode:  sslMode,
	}, nil
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode)
}

func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		value = strings.Trim(value, `"'`)

		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
