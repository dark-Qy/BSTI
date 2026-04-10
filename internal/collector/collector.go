package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"feishu-personality-agent/internal/persona"
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

	chatArgs := []string{"im", "+messages-search", "--start", startISO, "--end", endISO, "--page-size", "50", "--page-limit", "5", "--format", "json"}

	commands := []struct {
		domain string
		args   []string
	}{
		{"chat", chatArgs},
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

func BuildAnalysisPrompt(bundle Bundle, catalog []persona.Definition) string {
	var b strings.Builder
	b.WriteString("你是 BSPI（ByteStyle Personality Index）人格分析助手。不要做医学或心理诊断，不要展示思考过程。\n")
	b.WriteString("你必须只从给定的 20 个 BSPI 人格中选择 1 个最匹配的主人格，并只输出合法 JSON，不要输出 Markdown、解释文字或代码块外文本。\n")
	b.WriteString("JSON 字段必须严格为：primary_persona, summary, evidence, communication_style, work_preferences, blind_spots, confidence, disclaimer。\n")
	b.WriteString("其中 primary_persona 必须是给定的人格 shorthand 之一；evidence 必须是字符串数组。\n\n")
	if len(catalog) > 0 {
		b.WriteString("## BSPI Catalog\n")
		b.WriteString(persona.CompactPromptCatalog(catalog))
		b.WriteString("\n")
	}
	b.WriteString("## Authorized Data\n")
	for _, domain := range bundle.Domains {
		fmt.Fprintf(&b, "## %s\n", domain.Domain)
		if domain.Error != "" {
			fmt.Fprintf(&b, "error: %s\n", domain.Error)
		}
		b.WriteString(domain.Stdout)
		b.WriteString("\n\n")
	}
	b.WriteString("请基于授权数据完成 20 选 1，并输出 JSON。\n")
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
