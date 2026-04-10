package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AIDP         AIDPConfig
	Feishu       FeishuConfig
	LarkCLIBin   string
	AgentDataDir string
	HTTPAddr     string
}

type AIDPConfig struct {
	ModelHubURL string
	AK          string
	Model       string
	MaxTokens   int
	Stream      bool
}

type FeishuConfig struct {
	AppID     string
	AppSecret string
}

func Load(path string) (Config, error) {
	values := map[string]string{}
	file, err := os.Open(path)
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			values[strings.TrimSpace(key)] = trimEnvValue(value)
		}
		if err := scanner.Err(); err != nil {
			return Config{}, err
		}
	} else if !os.IsNotExist(err) {
		return Config{}, err
	}

	get := func(key, fallback string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		if value := values[key]; value != "" {
			return value
		}
		return fallback
	}

	maxTokens, err := strconv.Atoi(get("AIDP_MAX_TOKENS", "500"))
	if err != nil || maxTokens <= 0 {
		maxTokens = 500
	}
	stream, _ := strconv.ParseBool(get("AIDP_STREAM", "false"))

	return Config{
		AIDP: AIDPConfig{
			ModelHubURL: get("AIDP_MODELHUB_URL", "https://aidp.bytedance.net/api/modelhub/online/v2/crawl"),
			AK:          get("AIDP_AK", ""),
			Model:       get("AIDP_MODEL", "gpt-5.4-2026-03-05"),
			MaxTokens:   maxTokens,
			Stream:      stream,
		},
		Feishu: FeishuConfig{
			AppID:     get("LARK_APP_ID", ""),
			AppSecret: get("LARK_APP_SECRET", ""),
		},
		LarkCLIBin:   get("LARK_CLI_BIN", "lark-cli"),
		AgentDataDir: get("AGENT_DATA_DIR", "./data"),
		HTTPAddr:     get("HTTP_ADDR", "127.0.0.1:8787"),
	}, nil
}

func trimEnvValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
