package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	APIKey string `json:"api_key"`
}

// GetConfigPath returns the absolute path to the local config file (~/.okf/config.json).
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".okf", "config.json"), nil
}

// LoadConfig reads the config file from disk. If it doesn't exist, returns an empty config.
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	return &cfg, nil
}

// SaveConfig writes the config to disk with restricted permissions (0600).
func SaveConfig(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("could not serialize config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}

	return nil
}

// GetAPIKey resolves the API key in the following order:
// 1. Provided flag/variable (if not empty)
// 2. OKF_HUB_API_KEY environment variable
// 3. ~/.okf/config.json
func GetAPIKey(providedKey string) string {
	if providedKey != "" {
		return providedKey
	}

	envKey := os.Getenv("OKF_HUB_API_KEY")
	if envKey != "" {
		return envKey
	}

	cfg, err := LoadConfig()
	if err == nil && cfg.APIKey != "" {
		return cfg.APIKey
	}

	return ""
}
