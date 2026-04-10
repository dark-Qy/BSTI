package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsDotenvAndAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	err := os.WriteFile(envPath, []byte("AIDP_AK=test-ak\nLARK_APP_ID=cli_test\nLARK_APP_SECRET=secret\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.AIDP.ModelHubURL != "https://aidp.bytedance.net/api/modelhub/online/v2/crawl" {
		t.Fatalf("ModelHubURL = %q", cfg.AIDP.ModelHubURL)
	}
	if cfg.AIDP.AK != "test-ak" {
		t.Fatalf("AK = %q", cfg.AIDP.AK)
	}
	if cfg.Feishu.AppID != "cli_test" || cfg.Feishu.AppSecret != "secret" {
		t.Fatalf("Feishu config = %#v", cfg.Feishu)
	}
	if cfg.AIDP.MaxTokens != 5000 {
		t.Fatalf("MaxTokens = %d", cfg.AIDP.MaxTokens)
	}
}

func TestLoadDoesNotRequireSecretsForTests(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != "127.0.0.1:8787" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.AIDP.MaxTokens != 5000 {
		t.Fatalf("MaxTokens = %d", cfg.AIDP.MaxTokens)
	}
}

func TestLoadClampsConfiguredMaxTokensToMinimum(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	err := os.WriteFile(envPath, []byte("AIDP_MAX_TOKENS=1200\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AIDP.MaxTokens != 5000 {
		t.Fatalf("MaxTokens = %d", cfg.AIDP.MaxTokens)
	}
}

func TestLoadKeepsLargerConfiguredMaxTokens(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	err := os.WriteFile(envPath, []byte("AIDP_MAX_TOKENS=12000\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AIDP.MaxTokens != 12000 {
		t.Fatalf("MaxTokens = %d", cfg.AIDP.MaxTokens)
	}
}
