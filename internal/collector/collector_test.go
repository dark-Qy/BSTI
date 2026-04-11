package collector

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"feishu-personality-agent/internal/persona"
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

type scriptedRunner struct {
	calls [][]string
}

func (s *scriptedRunner) Run(ctx context.Context, args []string) (sandbox.Output, error) {
	s.calls = append(s.calls, append([]string(nil), args...))
	switch {
	case len(args) >= 2 && args[0] == "docs" && args[1] == "+search":
		return sandbox.Output{Stdout: `{"items":[{"url":"https://example.feishu.cn/docx/alpha"},{"url":"https://example.feishu.cn/docx/beta"}]}`}, nil
	case len(args) >= 2 && args[0] == "docs" && args[1] == "+fetch":
		return sandbox.Output{Stdout: `{"content":"doc body"}`}, nil
	case len(args) >= 2 && args[0] == "vc" && args[1] == "+search":
		return sandbox.Output{Stdout: `{"items":[{"calendar_event_id":"evt-1"},{"calendar_event_id":"evt-2"}]}`}, nil
	case len(args) >= 2 && args[0] == "vc" && args[1] == "+notes":
		return sandbox.Output{Stdout: `{"notes":"meeting notes"}`}, nil
	case len(args) >= 2 && args[0] == "mail" && args[1] == "+triage":
		return sandbox.Output{Stdout: `{"items":[{"message_id":"mid-1"},{"message_id":"mid-2"}]}`}, nil
	case len(args) >= 2 && args[0] == "mail" && args[1] == "+message":
		return sandbox.Output{Stdout: `{"body":"mail body"}`}, nil
	default:
		return sandbox.Output{Stdout: `{"ok":true}`}, nil
	}
}

func TestCollectFetchesDocBodiesMeetingNotesAndMailBodies(t *testing.T) {
	dir := t.TempDir()
	runner := &scriptedRunner{}
	c := New(runner)

	bundle, err := c.Collect(context.Background(), dir, time.Date(2026, 4, 10, 12, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Domains) != 11 {
		t.Fatalf("domains = %d", len(bundle.Domains))
	}

	var sawDocFetch, sawVCNotes, sawMailMessage bool
	for _, call := range runner.calls {
		if len(call) >= 2 && call[0] == "docs" && call[1] == "+fetch" {
			sawDocFetch = true
		}
		if len(call) >= 2 && call[0] == "vc" && call[1] == "+notes" {
			sawVCNotes = true
		}
		if len(call) >= 2 && call[0] == "mail" && call[1] == "+message" {
			sawMailMessage = true
		}
	}
	if !sawDocFetch {
		t.Fatal("expected docs +fetch")
	}
	if !sawVCNotes {
		t.Fatal("expected vc +notes")
	}
	if !sawMailMessage {
		t.Fatal("expected mail +message")
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

func TestBuildAnalysisPromptPreservesOriginalData(t *testing.T) {
	bundle := Bundle{Domains: []DomainResult{
		{Domain: "chat", Stdout: strings.Repeat("a", 2000), Error: strings.Repeat("z", 600)},
		{Domain: "docs", Stdout: strings.Repeat("b", 2000)},
		{Domain: "calendar", Stdout: strings.Repeat("c", 2000)},
		{Domain: "task", Stdout: strings.Repeat("d", 2000)},
		{Domain: "mail", Stdout: strings.Repeat("e", 2000)},
		{Domain: "vc", Stdout: strings.Repeat("f", 2000)},
	}}

	prompt := BuildAnalysisPrompt(bundle, persona.All())
	for _, want := range []string{
		"不要展示思考过程",
		"只输出合法 JSON",
		"primary_persona",
		"## BSPI Catalog",
		"## Authorized Data",
		"PRISM",
		"变色龙",
		"summary 需要写成一段信息密度高的中文总结",
		"evidence 至少提供 4 条",
		"优先引用跨域一致信号",
		"如果证据不足",
		"不要编造没有出现在授权数据中的事实",
		"避免使用带评判色彩或过度拟人化的措辞",
		"confidence 必须是 0 到 1 之间的数字",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing instruction %q", want)
		}
	}
	for _, domain := range []string{"chat", "docs", "calendar", "task", "mail", "vc"} {
		if !strings.Contains(prompt, "## "+domain) {
			t.Fatalf("prompt missing domain %s", domain)
		}
	}
	if !strings.Contains(prompt, "error: "+strings.Repeat("z", 600)) {
		t.Fatal("prompt should keep full error details")
	}
	if !strings.Contains(prompt, strings.Repeat("a", 2000)) || !strings.Contains(prompt, strings.Repeat("f", 2000)) {
		t.Fatal("prompt should keep full domain stdout")
	}
	if strings.Contains(prompt, "[truncated]") {
		t.Fatal("prompt should not add truncation markers")
	}
}
