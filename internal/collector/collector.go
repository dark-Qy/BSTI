package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

		switch command.domain {
		case "docs":
			extras, extraErr := c.collectDocBodies(ctx, out.Stdout)
			if extraErr != nil {
				bundle.Domains = append(bundle.Domains, DomainResult{
					Domain: "docs_content",
					Error:  extraErr.Error(),
				})
				continue
			}
			bundle.Domains = append(bundle.Domains, extras...)
		case "vc":
			extras, extraErr := c.collectVCNotes(ctx, out.Stdout)
			if extraErr != nil {
				bundle.Domains = append(bundle.Domains, DomainResult{
					Domain: "vc_notes",
					Error:  extraErr.Error(),
				})
				continue
			}
			bundle.Domains = append(bundle.Domains, extras...)
		case "mail":
			extras, extraErr := c.collectMailBodies(ctx, out.Stdout)
			if extraErr != nil {
				bundle.Domains = append(bundle.Domains, DomainResult{
					Domain: "mail_content",
					Error:  extraErr.Error(),
				})
				continue
			}
			bundle.Domains = append(bundle.Domains, extras...)
		}
	}
	return bundle, nil
}

func BuildAnalysisPrompt(bundle Bundle, catalog []persona.Definition) string {
	var b strings.Builder
	b.WriteString("你是 BSPI（ByteStyle Personality Index）人格分析助手。不要做医学或心理诊断，不要展示思考过程。\n")
	b.WriteString("你必须只从给定的 20 个 BSPI 人格中选择 1 个最匹配的主人格，并只输出合法 JSON，不要输出 Markdown、解释文字或代码块外文本。\n")
	b.WriteString("JSON 字段必须严格为：primary_persona, summary, evidence, communication_style, work_preferences, blind_spots, confidence, disclaimer。\n")
	b.WriteString("其中 primary_persona 必须是给定的人格 shorthand 之一；evidence 必须是字符串数组。\n\n")
	b.WriteString("写作要求：\n")
	b.WriteString("- summary 需要写成一段信息密度高的中文总结，不少于 120 字；要说明主人格判断、最关键的行为模式、跨场景一致性，以及结论成立的前提。\n")
	b.WriteString("- evidence 至少提供 4 条，优先引用跨域一致信号；每条都要尽量说明信号来自哪些授权数据域，例如 chat/docs/calendar/task/mail/vc。\n")
	b.WriteString("- communication_style 不能只写性格标签，要说明对方通常怎么表达、怎么推进讨论、在冲突或分歧时更可能怎样反应。\n")
	b.WriteString("- work_preferences 需要覆盖工作节奏、协作方式、决策偏好、任务选择或信息处理习惯，尽量给出稳定倾向而不是单次事件。\n")
	b.WriteString("- blind_spots 需要写得具体，说明潜在风险、容易被误解的点，以及什么情境下这些风险更容易出现。\n")
	b.WriteString("- confidence 必须是 0 到 1 之间的数字，并保留两位小数；数值越高代表证据越充分、跨域一致性越强。如果证据不足、数据偏单一或域之间相互矛盾，就降低分数。\n")
	b.WriteString("- disclaimer 需要强调这只是基于已授权工作数据的行为风格分析，不是能力评估、价值判断或医学诊断。\n")
	b.WriteString("- 不要编造没有出现在授权数据中的事实；如果某个维度缺乏证据，就明确说缺证据，不要硬补。\n\n")
	b.WriteString("- 避免使用带评判色彩或过度拟人化的措辞，保持克制、具体、证据导向的分析口吻。\n\n")
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

func (c *Collector) collectDocBodies(ctx context.Context, stdout string) ([]DomainResult, error) {
	refs := takeFirst(uniqueStrings(extractStringValues(stdout, func(key, value string) bool {
		key = strings.ToLower(key)
		if strings.Contains(value, "/docx/") || strings.Contains(value, "/wiki/") || strings.Contains(value, "/docs/") {
			return true
		}
		return (strings.Contains(key, "url") || strings.Contains(key, "doc")) && strings.HasPrefix(value, "http")
	})), 2)
	return c.runExtraFetches(ctx, "docs_content", refs, func(ref string) []string {
		return []string{"docs", "+fetch", "--doc", ref, "--format", "json"}
	})
}

func (c *Collector) collectVCNotes(ctx context.Context, stdout string) ([]DomainResult, error) {
	calendarIDs := takeFirst(uniqueStrings(extractStringValues(stdout, func(key, _ string) bool {
		key = strings.ToLower(key)
		return strings.Contains(key, "calendar_event_id")
	})), 3)
	if len(calendarIDs) > 0 {
		args := []string{"vc", "+notes", "--calendar-event-ids", strings.Join(calendarIDs, ","), "--format", "json"}
		out, err := c.runner.Run(ctx, args)
		return []DomainResult{{
			Domain:  "vc_notes",
			Command: strings.Join(args, " "),
			Stdout:  out.Stdout,
			Stderr:  out.Stderr,
			Error:   errorString(err),
		}}, nil
	}
	meetingIDs := takeFirst(uniqueStrings(extractStringValues(stdout, func(key, _ string) bool {
		key = strings.ToLower(key)
		return key == "meeting_id" || key == "meetingid"
	})), 3)
	if len(meetingIDs) > 0 {
		args := []string{"vc", "+notes", "--meeting-ids", strings.Join(meetingIDs, ","), "--format", "json"}
		out, err := c.runner.Run(ctx, args)
		return []DomainResult{{
			Domain:  "vc_notes",
			Command: strings.Join(args, " "),
			Stdout:  out.Stdout,
			Stderr:  out.Stderr,
			Error:   errorString(err),
		}}, nil
	}
	return nil, nil
}

func (c *Collector) collectMailBodies(ctx context.Context, stdout string) ([]DomainResult, error) {
	ids := takeFirst(uniqueStrings(extractStringValues(stdout, func(key, _ string) bool {
		key = strings.ToLower(key)
		return strings.Contains(key, "message_id")
	})), 2)
	return c.runExtraFetches(ctx, "mail_content", ids, func(id string) []string {
		return []string{"mail", "+message", "--message-id", id, "--format", "json"}
	})
}

func (c *Collector) runExtraFetches(ctx context.Context, domain string, refs []string, buildArgs func(string) []string) ([]DomainResult, error) {
	results := make([]DomainResult, 0, len(refs))
	for _, ref := range refs {
		args := buildArgs(ref)
		out, err := c.runner.Run(ctx, args)
		results = append(results, DomainResult{
			Domain:  domain,
			Command: strings.Join(args, " "),
			Stdout:  out.Stdout,
			Stderr:  out.Stderr,
			Error:   errorString(err),
		})
	}
	return results, nil
}

func extractStringValues(raw string, keep func(key, value string) bool) []string {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}
	var out []string
	var walk func(node any, parentKey string)
	walk = func(node any, parentKey string) {
		switch typed := node.(type) {
		case map[string]any:
			for key, value := range typed {
				walk(value, key)
			}
		case []any:
			for _, item := range typed {
				walk(item, parentKey)
			}
		case string:
			if keep(parentKey, typed) {
				out = append(out, strings.TrimSpace(typed))
			}
		}
	}
	walk(payload, "")
	return out
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func takeFirst(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
