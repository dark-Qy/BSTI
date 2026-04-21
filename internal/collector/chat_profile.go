package collector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func loadSessionUserOpenID(sessionDir string) (string, error) {
	path := filepath.Join(sessionDir, "lark-cli", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read session lark-cli config: %w", err)
	}

	var cfg struct {
		Apps []struct {
			Users []struct {
				UserOpenID string `json:"userOpenId"`
			} `json:"users"`
		} `json:"apps"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parse session lark-cli config: %w", err)
	}

	for _, app := range cfg.Apps {
		for _, user := range app.Users {
			if openID := strings.TrimSpace(user.UserOpenID); openID != "" {
				return openID, nil
			}
		}
	}
	return "", fmt.Errorf("no user open id found in session lark-cli config")
}
