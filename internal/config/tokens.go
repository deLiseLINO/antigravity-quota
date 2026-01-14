package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type TokenFile struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Expired      string `json:"expired"`
	ExpiresIn    int    `json:"expires_in"`
	Type         string `json:"type"`
	Email        string `json:"email"`
	FilePath     string `json:"-"` // Path to the file for saving updates
}

type TokenRefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

func LoadAllTokens() ([]*TokenFile, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".cli-proxy-api")
	entries, err := os.ReadDir(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*TokenFile{}, nil
		}
		return nil, fmt.Errorf("failed to read directory %s: %w", configDir, err)
	}

	var tokens []*TokenFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.Contains(strings.ToLower(name), "antigravity") && strings.HasSuffix(name, ".json") {
			filePath := filepath.Join(configDir, name)
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			var tokenFile TokenFile
			if err := json.Unmarshal(data, &tokenFile); err != nil {
				continue
			}

			tokenFile.FilePath = filePath
			if tokenFile.AccessToken != "" {
				tokens = append(tokens, &tokenFile)
			}
		}
	}

	return tokens, nil
}

func SaveToken(tf *TokenFile) error {
	if tf.FilePath == "" {
		return fmt.Errorf("file path not set in TokenFile")
	}

	data, err := json.MarshalIndent(tf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token file: %w", err)
	}

	if err := os.WriteFile(tf.FilePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

func SanitizeEmail(email string) string {
	if email == "" {
		return "default"
	}
	return strings.ReplaceAll(strings.ReplaceAll(email, "@", "_at_"), ".", "_dot_")
}

func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".cli-proxy-api"), nil
}

func EnsureConfigDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}
