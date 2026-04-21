package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExecutorAllowsOnlyConfiguredLarkCLICommands(t *testing.T) {
	exe := NewExecutor("lark-cli", t.TempDir(), time.Second)

	if err := exe.Validate([]string{"im", "+messages-search", "--format", "json"}); err != nil {
		t.Fatalf("expected im search to be allowed: %v", err)
	}
	if err := exe.Validate([]string{"im", "+chat-messages-list", "--chat-id", "oc_123", "--format", "json"}); err != nil {
		t.Fatalf("expected im chat message list to be allowed: %v", err)
	}
	if err := exe.Validate([]string{"im", "+chat-search", "--query", "project", "--format", "json"}); err != nil {
		t.Fatalf("expected im chat search to be allowed: %v", err)
	}
	if err := exe.Validate([]string{"im", "chat.members", "get", "--params", `{"chat_id":"oc_123"}`, "--format", "json"}); err != nil {
		t.Fatalf("expected im chat members get to be allowed: %v", err)
	}
	if err := exe.Validate([]string{"im", "chats", "get", "--params", `{"chat_id":"oc_123"}`, "--format", "json"}); err != nil {
		t.Fatalf("expected im chats get to be allowed: %v", err)
	}
	if err := exe.Validate([]string{"contact", "+get-user", "--user-id", "ou_123", "--format", "json"}); err != nil {
		t.Fatalf("expected contact get-user to be allowed: %v", err)
	}
	if err := exe.Validate([]string{"api", "POST", "/open-apis/im/v1/messages"}); err == nil {
		t.Fatal("expected raw api command to be rejected")
	}
	if err := exe.Validate([]string{"im", "+messages-send", "--chat-id", "oc_123", "--text", "hi"}); err == nil {
		t.Fatal("expected im write command to be rejected")
	}
	if err := exe.Validate([]string{"auth", "login", "--device-code", "abc"}); err != nil {
		t.Fatalf("expected auth device-code to be allowed: %v", err)
	}
	if err := exe.Validate([]string{"config", "init", "--new"}); err != nil {
		t.Fatalf("expected config init --new to be allowed: %v", err)
	}
}

func TestExecutorSetsPerSessionConfigDir(t *testing.T) {
	dir := t.TempDir()

	exe := NewExecutor(os.Args[0], dir, time.Second).WithEnv("FAKE_LARK_CLI=1")
	out, err := exe.Run(context.Background(), []string{"auth", "status"})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "lark-cli")
	if out.Stdout != want {
		t.Fatalf("config dir = %q, want %q", out.Stdout, want)
	}
}

func TestMain(m *testing.M) {
	if os.Getenv("FAKE_LARK_CLI") == "1" {
		_, _ = os.Stdout.WriteString(os.Getenv("LARKSUITE_CLI_CONFIG_DIR"))
		os.Exit(0)
	}
	os.Exit(m.Run())
}
