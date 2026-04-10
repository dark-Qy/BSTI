package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"feishu-personality-agent/internal/sandbox"
)

type Runner interface {
	Run(context.Context, []string) (sandbox.Output, error)
}

type Collector struct {
	runner Runner
}

type Bundle struct {
	Domains []DomainResult `json:"domains"`
}

type DomainResult struct {
	Domain  string `json:"domain"`
	Command string `json:"command"`
	Stdout  string `json:"stdout,omitempty"`
	Stderr  string `json:"stderr,omitempty"`
	Error   string `json:"error,omitempty"`
}

func New(runner Runner) *Collector {
	return &Collector{runner: runner}
}

func (c *Collector) Collect(ctx context.Context, sessionDir string, now time.Time) (Bundle, error) {
	logDir := filepath.Join(sessionDir, "logs")
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return Bundle{}, err
	}

	start := now.AddDate(0, 0, -30)
	startDate := start.Format("2006-01-02")
	endDate := now.Format("2006-01-02")
	startISO := start.Format("2006-01-02T15:04:05-07:00")
	endISO := now.Format("2006-01-02T15:04:05-07:00")

	j, _ := json.Marshal(map[string]any{
		"open_time": map[string]string{"start": startISO, "end": endISO},
	})
	mailFilter, _ := json.Marshal(map[string]any{
		"time_range": map[string]string{"start_time": startISO, "end_time": endISO},
	})

	commands := []struct {
		domain string
		args   []string
	}{
		{"chat", []string{"im", "+messages-search", "--query", "", "--start", startISO, "--end", endISO, "--page-size", "50", "--page-limit", "5", "--format", "json"}},
		{"docs", []string{"docs", "+search", "--filter", string(j), "--page-size", "20", "--format", "json"}},
		{"calendar", []string{"calendar", "+agenda", "--start", startDate, "--end", endDate, "--format", "json"}},
		{"task", []string{"task", "+get-my-tasks", "--created_at", "-30d", "--page-limit", "5", "--format", "json"}},
		{"mail", []string{"mail", "+triage", "--filter", string(mailFilter), "--max", "100", "--format", "json"}},
		{"vc", []string{"vc", "+search", "--start", startDate, "--end", endDate, "--page-size", "30", "--format", "json"}},
	}

	bundle := Bundle{Domains: make([]DomainResult, 0, len(commands))}
	for _, command := range commands {
		result := DomainResult{Domain: command.domain, Command: strings.Join(command.args, " ")}
		out, err := c.runner.Run(ctx, command.args)
		result.Stdout = out.Stdout
		result.Stderr = out.Stderr
		if err != nil {
			result.Error = err.Error()
		}
		if writeErr := appendRaw(filepath.Join(logDir, "raw-"+command.domain+".jsonl"), result); writeErr != nil {
			return bundle, writeErr
		}
		bundle.Domains = append(bundle.Domains, result)
	}
	return bundle, nil
}

func BuildAnalysisPrompt(bundle Bundle) string {
	var b strings.Builder
	b.WriteString("你是一个谨慎的人格行为分析助手。基于用户授权的飞书数据，生成 MBTI-like 行为风格报告。不要做医学或心理诊断。\n\n")
	for _, domain := range bundle.Domains {
		fmt.Fprintf(&b, "## Domain: %s\n", domain.Domain)
		if domain.Error != "" {
			fmt.Fprintf(&b, "Collection error: %s\n", domain.Error)
		}
		text := domain.Stdout
		if len(text) > 12000 {
			text = text[:12000] + "\n...[truncated]"
		}
		b.WriteString(text)
		b.WriteString("\n\n")
	}
	b.WriteString("请输出 Markdown，包含：人格倾向、证据、沟通风格、工作偏好、风险/盲区、置信度、免责声明。\n")
	return b.String()
}

func appendRaw(path string, result DomainResult) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(result)
}
