package config

import (
	"bufio"
	"os"
	"strings"
)

// loadDotEnv reads a .env file and sets any key=value pairs as environment variables,
// skipping keys that are already set in the environment.
func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return // .env is optional; silently skip if absent
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}
