package config

import "os"

type Config struct {
	DBPath string
	Port   string
}

func Load() Config {
	loadDotEnv(".env")
	return Config{
		DBPath: getEnv("DB_PATH", "./data.db"),
		Port:   getEnv("PORT", ":8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
