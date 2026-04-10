package collector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"feishu-personality-agent/internal/sandbox"
)

type fakeRunner struct {
	calls [][]string
}

func (f *fakeRunner) Run(ctx context.Context, args []string) (sandbox.Output, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	return sandbox.Output{Stdout: `{"ok":true}`}, nil
}

func TestCollectWritesPrivateRawLogsForCoreDomains(t *testing.T) {
	dir := t.TempDir()
	runner := &fakeRunner{}
	c := New(runner)

	bundle, err := c.Collect(context.Background(), dir, time.Date(2026, 4, 10, 12, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Domains) != 6 {
		t.Fatalf("domains = %d", len(bundle.Domains))
	}
	for _, domain := range []string{"chat", "docs", "calendar", "task", "mail", "vc"} {
		path := filepath.Join(dir, "logs", "raw-"+domain+".jsonl")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing raw log %s: %v", domain, err)
		}
	}
	if len(runner.calls) != 6 {
		t.Fatalf("calls = %d", len(runner.calls))
	}
}

func TestCollectOmitsEmptyChatQueryArgument(t *testing.T) {
	dir := t.TempDir()
	runner := &fakeRunner{}
	c := New(runner)

	if _, err := c.Collect(context.Background(), dir, time.Date(2026, 4, 10, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) == 0 {
		t.Fatal("no collector calls recorded")
	}
	chatArgs := runner.calls[0]
	for i, arg := range chatArgs {
		if arg == "" {
			t.Fatalf("chat arg %d is empty: %#v", i, chatArgs)
		}
		if arg == "--query" {
			t.Fatalf("chat args include --query for blank search: %#v", chatArgs)
		}
	}
}
