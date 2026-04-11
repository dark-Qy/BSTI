package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultLLMMaxTokens = 5000

type Provider string

const (
	ProviderModelHub Provider = "modelhub"
	ProviderKimi     Provider = "kimi"
)

type Config struct {
	LLM          LLMConfig
	Feishu       FeishuConfig
	LarkCLIBin   string
	AgentDataDir string
	HTTPAddr     string
}

type LLMConfig struct {
	Provider  Provider
	APIURL    string
	APIKey    string
	Model     string
	MaxTokens int
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

	provider := Provider(strings.ToLower(get("LLM_PROVIDER", "")))
	if provider == "" {
		return Config{}, fmt.Errorf("missing LLM_PROVIDER in .env")
	}
	if provider != ProviderModelHub && provider != ProviderKimi {
		return Config{}, fmt.Errorf("invalid LLM_PROVIDER %q", provider)
	}

	maxTokens, err := strconv.Atoi(get("LLM_MAX_TOKENS", "5000"))
	if err != nil || maxTokens <= 0 {
		maxTokens = defaultLLMMaxTokens
	}
	if maxTokens < defaultLLMMaxTokens {
		maxTokens = defaultLLMMaxTokens
	}

	apiURL := get("LLM_API_URL", "")
	if apiURL == "" {
		return Config{}, fmt.Errorf("missing LLM_API_URL in .env")
	}
	apiKey := get("LLM_API_KEY", "")
	if apiKey == "" {
		return Config{}, fmt.Errorf("missing LLM_API_KEY in .env")
	}
	model := get("LLM_MODEL", "")
	if model == "" {
		return Config{}, fmt.Errorf("missing LLM_MODEL in .env")
	}

	return Config{
		LLM: LLMConfig{
			Provider:  provider,
			APIURL:    apiURL,
			APIKey:    apiKey,
			Model:     model,
			MaxTokens: maxTokens,
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
