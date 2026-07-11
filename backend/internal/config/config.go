package config

import (
	"os"
	"time"
)

type Config struct {
	DatabaseURL  string
	HTTPAddr     string
	BotToken     string
	AuthTTL      time.Duration
	TMDBToken    string
	TMDBBaseURL  string
	TMDBLanguage string
}

func Load() Config {
	port := getenv("HTTP_PORT", "8080")
	authTTL := getenv("AUTH_TTL", "24h")
	parsedTTL, err := time.ParseDuration(authTTL)
	if err != nil {
		parsedTTL = 24 * time.Hour
	}

	return Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		HTTPAddr:     ":" + port,
		BotToken:     os.Getenv("BOT_TOKEN"),
		AuthTTL:      parsedTTL,
		TMDBToken:    os.Getenv("TMDB_API_TOKEN"),
		TMDBBaseURL:  getenv("TMDB_BASE_URL", "https://api.themoviedb.org/3"),
		TMDBLanguage: getenv("TMDB_LANGUAGE", "ru-RU"),
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
