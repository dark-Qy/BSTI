package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsDotenvAndAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	err := os.WriteFile(envPath, []byte("LLM_PROVIDER=modelhub\nLLM_API_URL=https://modelhub.example/api\nLLM_API_KEY=test-key\nLLM_MODEL=test-model\nLARK_APP_ID=cli_test\nLARK_APP_SECRET=secret\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.LLM.Provider != ProviderModelHub {
		t.Fatalf("Provider = %q", cfg.LLM.Provider)
	}
	if cfg.LLM.APIURL != "https://modelhub.example/api" {
		t.Fatalf("APIURL = %q", cfg.LLM.APIURL)
	}
	if cfg.LLM.APIKey != "test-key" {
		t.Fatalf("APIKey = %q", cfg.LLM.APIKey)
	}
	if cfg.LLM.Model != "test-model" {
		t.Fatalf("Model = %q", cfg.LLM.Model)
	}
	if cfg.Feishu.AppID != "cli_test" || cfg.Feishu.AppSecret != "secret" {
		t.Fatalf("Feishu config = %#v", cfg.Feishu)
	}
	if cfg.LLM.MaxTokens != 5000 {
		t.Fatalf("MaxTokens = %d", cfg.LLM.MaxTokens)
	}
}

func TestLoadRequiresProvider(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	err := os.WriteFile(envPath, []byte("LLM_API_URL=https://modelhub.example/api\nLLM_API_KEY=test-key\nLLM_MODEL=test-model\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Load(envPath)
	if err == nil {
		t.Fatal("expected missing provider error")
	}
	if err.Error() != "missing LLM_PROVIDER or CUSTOM_LLM_PROVIDER in env or .env" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestLoadRejectsUnknownProvider(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	err := os.WriteFile(envPath, []byte("LLM_PROVIDER=custom\nLLM_API_URL=https://example.com/api\nLLM_API_KEY=test-key\nLLM_MODEL=test-model\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Load(envPath)
	if err == nil {
		t.Fatal("expected invalid provider error")
	}
	if err.Error() != `invalid LLM_PROVIDER "custom"` {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestLoadClampsConfiguredMaxTokensToMinimum(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	err := os.WriteFile(envPath, []byte("LLM_PROVIDER=kimi\nLLM_API_URL=https://api.moonshot.cn/v1/chat/completions\nLLM_API_KEY=Bearer test-key\nLLM_MODEL=kimi-k2.5\nLLM_MAX_TOKENS=1200\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLM.MaxTokens != 5000 {
		t.Fatalf("MaxTokens = %d", cfg.LLM.MaxTokens)
	}
}

func TestLoadKeepsLargerConfiguredMaxTokens(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	err := os.WriteFile(envPath, []byte("LLM_PROVIDER=kimi\nLLM_API_URL=https://api.moonshot.cn/v1/chat/completions\nLLM_API_KEY=Bearer test-key\nLLM_MODEL=kimi-k2.5\nLLM_MAX_TOKENS=12000\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLM.MaxTokens != 12000 {
		t.Fatalf("MaxTokens = %d", cfg.LLM.MaxTokens)
	}
}

func TestLoadFallsBackToCustomLLMEnvAndFaaSDefaults(t *testing.T) {
	t.Setenv("CUSTOM_LLM_PROVIDER", "kimi")
	t.Setenv("CUSTOM_LLM_API_URL", "https://api.moonshot.cn/v1/chat/completions")
	t.Setenv("CUSTOM_LLM_API_KEY", "Bearer custom-test-key")
	t.Setenv("CUSTOM_LLM_MODEL", "kimi-k2.5")
	t.Setenv("CUSTOM_LLM_MAX_TOKENS", "9000")

	cfg, err := Load(filepath.Join(t.TempDir(), ".env"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.LLM.Provider != ProviderKimi {
		t.Fatalf("Provider = %q", cfg.LLM.Provider)
	}
	if cfg.LLM.APIURL != "https://api.moonshot.cn/v1/chat/completions" {
		t.Fatalf("APIURL = %q", cfg.LLM.APIURL)
	}
	if cfg.LLM.APIKey != "Bearer custom-test-key" {
		t.Fatalf("APIKey = %q", cfg.LLM.APIKey)
	}
	if cfg.LLM.Model != "kimi-k2.5" {
		t.Fatalf("Model = %q", cfg.LLM.Model)
	}
	if cfg.LLM.MaxTokens != 9000 {
		t.Fatalf("MaxTokens = %d", cfg.LLM.MaxTokens)
	}
	if cfg.AgentDataDir != "/tmp/byte-agent-data" {
		t.Fatalf("AgentDataDir = %q", cfg.AgentDataDir)
	}
	if cfg.HTTPAddr != "0.0.0.0:8787" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
}

func TestLoadPrefersLLMEnvOverCustomFallback(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "modelhub")
	t.Setenv("LLM_API_URL", "https://modelhub.example/api")
	t.Setenv("LLM_API_KEY", "primary-key")
	t.Setenv("LLM_MODEL", "primary-model")
	t.Setenv("LLM_MAX_TOKENS", "7000")
	t.Setenv("CUSTOM_LLM_PROVIDER", "kimi")
	t.Setenv("CUSTOM_LLM_API_URL", "https://api.moonshot.cn/v1/chat/completions")
	t.Setenv("CUSTOM_LLM_API_KEY", "fallback-key")
	t.Setenv("CUSTOM_LLM_MODEL", "fallback-model")
	t.Setenv("CUSTOM_LLM_MAX_TOKENS", "9000")

	cfg, err := Load(filepath.Join(t.TempDir(), ".env"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.LLM.Provider != ProviderModelHub {
		t.Fatalf("Provider = %q", cfg.LLM.Provider)
	}
	if cfg.LLM.APIURL != "https://modelhub.example/api" {
		t.Fatalf("APIURL = %q", cfg.LLM.APIURL)
	}
	if cfg.LLM.APIKey != "primary-key" {
		t.Fatalf("APIKey = %q", cfg.LLM.APIKey)
	}
	if cfg.LLM.Model != "primary-model" {
		t.Fatalf("Model = %q", cfg.LLM.Model)
	}
	if cfg.LLM.MaxTokens != 7000 {
		t.Fatalf("MaxTokens = %d", cfg.LLM.MaxTokens)
	}
}
