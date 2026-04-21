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
	Domains       []DomainResult `json:"domains"`
	ChatSummary   ChatSummary    `json:"chat_summary"`
	DocsSummary   DocsSummary    `json:"docs_summary"`
	AnalysisInput AnalysisInput  `json:"analysis_input"`
}

type DomainResult struct {
	Domain  string `json:"domain"`
	Command string `json:"command"`
	Stdout  string `json:"stdout,omitempty"`
	Stderr  string `json:"stderr,omitempty"`
	Error   string `json:"error,omitempty"`
}

type mainDomainOutcome struct {
	Result      DomainResult
	ChatSummary ChatSummary
	ChatPrompt  string
	ChatStats   DomainDigestStats
	DocsSummary DocsSummary
	DocsRefs    []string
	MailStats   DomainDigestStats
}

type extraDomainOutcome struct {
	Domain  string
	Results []DomainResult
	Error   error
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

	mailFilter, _ := json.Marshal(map[string]any{
		"time_range": map[string]string{"start_time": startISO, "end_time": endISO},
	})

	mainJobs := []struct {
		domain string
		run    func(context.Context) (mainDomainOutcome, error)
	}{
		{
			domain: "chat",
			run: func(ctx context.Context) (mainDomainOutcome, error) {
				result, summary, promptData, stats, err := c.collectChatDomain(ctx, sessionDir, startISO, endISO)
				return mainDomainOutcome{Result: result, ChatSummary: summary, ChatPrompt: promptData, ChatStats: stats}, err
			},
		},
		{
			domain: "docs",
			run: func(ctx context.Context) (mainDomainOutcome, error) {
				result, summary, refs, err := c.searchDocsDomain(ctx, sessionDir, startISO, endISO)
				return mainDomainOutcome{Result: result, DocsSummary: summary, DocsRefs: refs}, err
			},
		},
		{
			domain: "calendar",
			run: func(ctx context.Context) (mainDomainOutcome, error) {
				return mainDomainOutcome{
					Result: c.runDomainCommand(ctx, "calendar", []string{"calendar", "+agenda", "--start", startDate, "--end", endDate, "--format", "json"}),
				}, nil
			},
		},
		{
			domain: "task",
			run: func(ctx context.Context) (mainDomainOutcome, error) {
				return mainDomainOutcome{
					Result: c.runDomainCommand(ctx, "task", []string{"task", "+get-my-tasks", "--created_at", "-30d", "--page-limit", "5", "--format", "json"}),
				}, nil
			},
		},
		{
			domain: "mail",
			run: func(ctx context.Context) (mainDomainOutcome, error) {
				result := c.runDomainCommand(ctx, "mail", []string{"mail", "+triage", "--filter", string(mailFilter), "--max", "100", "--format", "json"})
				filtered, stats := filterMailDomain(result)
				return mainDomainOutcome{
					Result:    filtered,
					MailStats: stats,
				}, nil
			},
		},
		{
			domain: "vc",
			run: func(ctx context.Context) (mainDomainOutcome, error) {
				return mainDomainOutcome{
					Result: c.runDomainCommand(ctx, "vc", []string{"vc", "+search", "--start", startDate, "--end", endDate, "--page-size", "30", "--format", "json"}),
				}, nil
			},
		},
	}

	type mainRunResult struct {
		outcome mainDomainOutcome
		err     error
	}

	mainRuns := runBoundedOrdered(ctx, mainJobs, mainDomainConcurrency, func(ctx context.Context, job struct {
		domain string
		run    func(context.Context) (mainDomainOutcome, error)
	}) mainRunResult {
		outcome, err := job.run(ctx)
		return mainRunResult{outcome: outcome, err: err}
	})

	mainByDomain := make(map[string]mainDomainOutcome, len(mainJobs))
	for index, job := range mainJobs {
		if mainRuns[index].err != nil {
			return Bundle{}, mainRuns[index].err
		}
		mainByDomain[job.domain] = mainRuns[index].outcome
	}

	extraJobs := []struct {
		domain string
		run    func(context.Context) ([]DomainResult, error)
	}{
		{
			domain: "docs_content",
			run: func(ctx context.Context) ([]DomainResult, error) {
				return c.runExtraFetches(ctx, "docs_content", mainByDomain["docs"].DocsRefs, func(ref string) []string {
					return []string{"docs", "+fetch", "--doc", ref, "--format", "json"}
				})
			},
		},
		{
			domain: "mail_content",
			run: func(ctx context.Context) ([]DomainResult, error) {
				return c.collectMailBodies(ctx, mainByDomain["mail"].Result.Stdout)
			},
		},
		{
			domain: "vc_notes",
			run: func(ctx context.Context) ([]DomainResult, error) {
				return c.collectVCNotes(ctx, mainByDomain["vc"].Result.Stdout)
			},
		},
	}

	extraRuns := runBoundedOrdered(ctx, extraJobs, extraFetchConcurrency, func(ctx context.Context, job struct {
		domain string
		run    func(context.Context) ([]DomainResult, error)
	}) extraDomainOutcome {
		results, err := job.run(ctx)
		return extraDomainOutcome{Domain: job.domain, Results: results, Error: err}
	})

	extraByDomain := make(map[string]extraDomainOutcome, len(extraJobs))
	for _, extra := range extraRuns {
		extraByDomain[extra.Domain] = extra
	}
	filteredMailResult, filteredMailContents := filterMailByBodies(mainByDomain["mail"].Result, appendExtraResults(extraByDomain["mail_content"], "mail_content"))
	mainByDomain["mail"] = mainDomainOutcome{
		Result:      filteredMailResult,
		MailStats:   mainByDomain["mail"].MailStats,
		ChatSummary: mainByDomain["mail"].ChatSummary,
		ChatPrompt:  mainByDomain["mail"].ChatPrompt,
		ChatStats:   mainByDomain["mail"].ChatStats,
		DocsSummary: mainByDomain["mail"].DocsSummary,
		DocsRefs:    mainByDomain["mail"].DocsRefs,
	}
	extraByDomain["mail_content"] = extraDomainOutcome{Domain: "mail_content", Results: filteredMailContents}

	bundle := Bundle{Domains: make([]DomainResult, 0, len(mainJobs)+6)}
	for _, domain := range []string{"chat", "docs", "calendar", "task", "mail", "vc"} {
		result := mainByDomain[domain].Result
		if writeErr := appendRaw(filepath.Join(logDir, "raw-"+domain+".jsonl"), result); writeErr != nil {
			return bundle, writeErr
		}
		switch domain {
		case "chat":
			bundle.ChatSummary = mainByDomain[domain].ChatSummary
			result.Stdout = mainByDomain[domain].ChatPrompt
			bundle.Domains = append(bundle.Domains, result)
		case "docs":
			bundle.DocsSummary = mainByDomain[domain].DocsSummary
			bundle.Domains = append(bundle.Domains, result)
			bundle.Domains = append(bundle.Domains, appendExtraResults(extraByDomain["docs_content"], "docs_content")...)
		case "mail":
			bundle.Domains = append(bundle.Domains, result)
			bundle.Domains = append(bundle.Domains, appendExtraResults(extraByDomain["mail_content"], "mail_content")...)
		case "vc":
			bundle.Domains = append(bundle.Domains, result)
			bundle.Domains = append(bundle.Domains, appendExtraResults(extraByDomain["vc_notes"], "vc_notes")...)
		default:
			bundle.Domains = append(bundle.Domains, result)
		}
	}
	bundle.AnalysisInput = BuildAnalysisInput(bundle, mainByDomain["chat"].ChatStats, mainByDomain["mail"].MailStats)
	return bundle, nil
}

func (c *Collector) runDomainCommand(ctx context.Context, domain string, args []string) DomainResult {
	out, err := c.runner.Run(ctx, args)
	return DomainResult{
		Domain:  domain,
		Command: strings.Join(args, " "),
		Stdout:  out.Stdout,
		Stderr:  out.Stderr,
		Error:   errorString(err),
	}
}

func appendExtraResults(outcome extraDomainOutcome, fallbackDomain string) []DomainResult {
	if outcome.Error == nil {
		return outcome.Results
	}
	return []DomainResult{{
		Domain: fallbackDomain,
		Error:  outcome.Error.Error(),
	}}
}

func BuildAnalysisPrompt(bundle Bundle, catalog []persona.Definition) string {
	var b strings.Builder
	b.WriteString("你是 BSPI（ByteStyle Personality Index）人格分析助手。不要做医学或心理诊断，不要展示思考过程。\n")
	b.WriteString("你必须只从给定的 20 个 BSPI 人格中选择 1 个最匹配的主人格，并只输出合法 JSON，不要输出 Markdown、解释文字或代码块外文本。\n")
	b.WriteString("请严格按以下顺序分析：阶段 1：行为事实提取；阶段 2：跨域交叉分析；阶段 3：人格匹配与个性化输出。\n")
	b.WriteString("阶段 1：行为事实提取。分别从 chat/docs/calendar/task/mail/vc 提取表达特征、工作模式、协作行为、知识信号，不要急于贴人格标签。\n")
	b.WriteString("优先从 docs_content 提取稳定的工作流、文档写作风格、知识积累、显式观点和决策线索。\n")
	b.WriteString("阶段 2：跨域交叉分析。识别跨域一致信号、跨域矛盾或反差点、仅单域出现的弱信号，并明确证据强弱。\n")
	b.WriteString("当 chat 与 docs 同时存在时，优先用 docs 作为长期稳定信号，chat 作为即时互动信号。\n")
	b.WriteString("固定跨域分析矩阵：\n")
	b.WriteString("- 协作方式：chat 的单聊/群聊推进方式，docs 的协作文档痕迹，calendar 的会议角色。\n")
	b.WriteString("- 决策路径：chat 中是否追问证据，docs 中是否写量化指标/方案约束，calendar/vc 中是否有评审或对齐。\n")
	b.WriteString("- 信息处理：chat 的响应方式，docs 的结构化程度，mail 的结论压缩能力。\n")
	b.WriteString("- 风险与推进：chat 中如何催动与承诺，docs 中如何写风险与边界，vc 中如何沉淀行动项。\n")
	b.WriteString("阶段 3：人格匹配与个性化输出。基于前两阶段结果完成 20 选 1，并给出结构化画像。\n")
	b.WriteString("反事实校验：在确定 primary_persona 后，必须检查最接近的第二候选人格，并在 summary 或 contrast_signals 中说明为什么最终排除第二候选。\n")
	b.WriteString("JSON 字段必须严格为：primary_persona, summary, evidence, work_profile, expression_fingerprint, output_style, knowledge_signals, communication_style, work_preferences, blind_spots, highlight_tags, behavior_vectors, interaction_insights, contrast_signals, confidence, disclaimer。\n")
	b.WriteString("其中 primary_persona 必须是给定的人格 shorthand 之一；evidence 必须是对象数组；highlight_tags 必须是字符串数组。\n")
	b.WriteString("work_profile 必须包含 responsibility_scope, typical_workflow, doc_writing_style, decision_making_pattern, tech_stack_or_domain。\n")
	b.WriteString("expression_fingerprint 必须包含 catchphrases, jargon, sentence_pattern, emoji_habit, formality_spectrum, reply_speed_pattern, conflict_expression。\n")
	b.WriteString("output_style 必须包含 doc_structure_preference, detail_level, email_reply_pattern, chat_reply_pattern, meeting_behavior。\n")
	b.WriteString("knowledge_signals 必须包含 explicit_opinions, learned_lessons, repeated_concerns, reference_sources。\n")
	b.WriteString("behavior_vectors 必须是长度为 6 的数组，且每个对象都必须包含 label, left_pole, right_pole, score, summary。\n")
	b.WriteString("interaction_insights 必须是对象，且包含 relationship_summary, core_collaborators, frequent_people, frequent_chats 四个字段；后三者是数组，数组元素必须包含 display_name, summary, evidence，且不要输出 open_id、chat_id、identifier 或其他内部标识。\n")
	b.WriteString("硬规则：frequent_chats 只能描述工作群、项目群、专题群等群聊信号；单聊、私聊、P2P、与某人 direct message 往返，一律写入 core_collaborators 或 frequent_people，绝对不要写入 frequent_chats。\n")
	b.WriteString("behavior_vectors 的顺序和极点必须严格如下：\n")
	b.WriteString("1. 协作方式 | 独立成局 | 高频协同\n")
	b.WriteString("2. 表达风格 | 克制压缩 | 高频输出\n")
	b.WriteString("3. 决策路径 | 证据校准 | 直觉快判\n")
	b.WriteString("4. 推进节奏 | 稳态推进 | 高压突进\n")
	b.WriteString("5. 信息处理 | 深度聚焦 | 广度扫描\n")
	b.WriteString("6. 风险态度 | 防御优先 | 进攻优先\n")
	b.WriteString("score 必须是 0 到 100 的整数；summary 需要解释该维度为何得到该分值。\n\n")
	b.WriteString("写作要求：\n")
	b.WriteString("- summary 需要写成一段信息密度高的中文总结，不少于 120 字；summary 必须包含最显著的区分性行为、跨场景一致性、支撑结论的关键证据，以及如果存在则写出跨域矛盾或反差点。\n")
	b.WriteString("- evidence 至少提供 4 条，优先引用跨域一致信号；evidence 必须是对象数组，每条都要包含 domains, behavior, strength, is_cross_domain, is_distinctive。\n")
	b.WriteString("- communication_style 不能只写性格标签，要说明对方通常怎么表达、怎么推进讨论、在冲突或分歧时更可能怎样反应。\n")
	b.WriteString("- work_preferences 需要覆盖工作节奏、协作方式、决策偏好、任务选择或信息处理习惯，尽量给出稳定倾向而不是单次事件。\n")
	b.WriteString("- 在 work_profile.doc_writing_style、knowledge_signals、work_preferences 等字段里优先引用文档证据；如果文档与聊天信号不一致，要明确指出差异。\n")
	b.WriteString("- blind_spots 需要写得具体，说明潜在风险、容易被误解的点，以及什么情境下这些风险更容易出现。\n")
	b.WriteString("- highlight_tags 需要给出 2 到 4 个短标签，适合结果页首屏快速感知，不要写成长句。\n")
	b.WriteString("- contrast_signals 可以提供 0 到 3 条短句，用于描述跨域矛盾、反差点或排除第二候选人格的关键信号。\n")
	b.WriteString("- confidence 必须是 0 到 1 之间的数字，并保留两位小数；数值越高代表证据越充分、跨域一致性越强。如果证据不足、数据偏单一或域之间相互矛盾，就降低分数。\n")
	b.WriteString("- disclaimer 需要强调这只是基于已授权工作数据的行为风格分析，不是能力评估、价值判断或医学诊断。\n")
	b.WriteString("- summary 必须包含至少 2 个“该用户独有或高度区分”的具体行为，不能退化成通用工作评价。\n")
	b.WriteString("- evidence 至少 2 条必须引用原话或原文片段，而不是纯概括。\n")
	b.WriteString("- behavior_vectors[*].summary 必须引用具体行为依据，不能只写抽象评语。\n")
	b.WriteString("- 若证据主要来自 docs，必须在 summary 或 evidence 中明确写出 docs 是主信号源。\n")
	b.WriteString("- 若 chat 信号稀薄或偏噪声，必须显式降权，不能硬凑人格结论。\n")
	b.WriteString("- 负面示例：不要把“与小李单聊”“direct message 高频往返”“P2P 一对一讨论”写进 frequent_chats，这些只能归入 core_collaborators 或 frequent_people。\n")
	b.WriteString("- 不要编造没有出现在授权数据中的事实；如果某个维度缺乏证据，就明确说缺证据，不要硬补。\n\n")
	b.WriteString("- 避免使用带评判色彩或过度拟人化的措辞，保持克制、具体、证据导向的分析口吻。\n\n")
	if len(catalog) > 0 {
		b.WriteString("## BSPI Catalog\n")
		b.WriteString(persona.CompactPromptCatalog(catalog))
		b.WriteString("\n")
	}
	if bundle.ChatSummary.RelationshipSummary != "" {
		b.WriteString("## interaction_insights_reference\n")
		raw, _ := json.Marshal(chatSummaryReference(bundle.ChatSummary))
		b.Write(raw)
		b.WriteString("\n\n")
	}
	if bundle.DocsSummary.SelectionRule != "" {
		b.WriteString("## SelectionRule\n")
		b.WriteString(bundle.DocsSummary.SelectionRule + "\n\n")
		b.WriteString("## docs_summary_reference\n")
		raw, _ := json.Marshal(bundle.DocsSummary)
		b.Write(raw)
		b.WriteString("\n\n")
	}
	b.WriteString("## Authorized Data\n")
	for _, domain := range bundle.Domains {
		switch domain.Domain {
		case "chat", "docs", "docs_content", "calendar", "task", "mail", "mail_content", "vc", "vc_notes":
			fmt.Fprintf(&b, "## %s\n", domain.Domain)
			if domain.Domain == "chat" {
				writeChatStatsSection(&b, bundle.AnalysisInput.Stats["chat_prompt"])
			}
			b.WriteString(sanitizePromptDomain(domain.Domain, domain.Stdout))
			b.WriteString("\n\n")
		}
	}
	b.WriteString("请基于授权数据完成 20 选 1，并输出 JSON。\n")
	return b.String()
}

func writeChatStatsSection(b *strings.Builder, stats DomainDigestStats) {
	if stats.ItemsBefore == 0 && stats.ItemsAfter == 0 && stats.FilteredItems == 0 && stats.FetchedItems == 0 {
		return
	}
	fmt.Fprintf(
		b,
		"chat_stats: 最近 30 天原始因果链上下文消息 %d 条，保留 %d 条，过滤 %d 条，涉及 %d 个会话。\n",
		stats.ItemsBefore,
		stats.ItemsAfter,
		stats.FilteredItems,
		stats.FetchedItems,
	)
}

func Coverage(bundle Bundle) persona.Coverage {
	coverage := persona.Coverage{
		SuccessfulDomains: []string{},
		FailedDomains:     []string{},
	}
	primaryDomains := map[string]struct{}{
		"chat":     {},
		"docs":     {},
		"calendar": {},
		"task":     {},
		"mail":     {},
		"vc":       {},
	}
	successSet := map[string]struct{}{}
	failureSet := map[string]struct{}{}
	for _, domain := range bundle.Domains {
		if _, ok := primaryDomains[domain.Domain]; !ok {
			continue
		}
		if strings.TrimSpace(domain.Error) == "" {
			successSet[domain.Domain] = struct{}{}
			continue
		}
		failureSet[domain.Domain] = struct{}{}
	}
	coverage.SuccessfulDomains = sortedKeys(successSet)
	coverage.FailedDomains = sortedKeys(failureSet)
	switch {
	case len(coverage.SuccessfulDomains) > 0 && len(coverage.FailedDomains) > 0:
		coverage.Summary = fmt.Sprintf("已覆盖 %d 个数据域，%d 个数据域因权限或命令失败未纳入。", len(coverage.SuccessfulDomains), len(coverage.FailedDomains))
	case len(coverage.SuccessfulDomains) > 0:
		coverage.Summary = fmt.Sprintf("已覆盖 %d 个数据域，授权数据足以生成当前结果。", len(coverage.SuccessfulDomains))
	default:
		coverage.Summary = "未获得可用数据域，当前结果可能不完整。"
	}
	return coverage
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

func chatSummaryReference(summary ChatSummary) ChatSummary {
	return ChatSummary{
		RelationshipSummary: summary.RelationshipSummary,
		CoreCollaborators:   scrubSummaryItems(summary.CoreCollaborators),
		FrequentPeople:      scrubSummaryItems(summary.FrequentPeople),
		FrequentChats:       scrubSummaryItems(summary.FrequentChats),
	}
}

func scrubSummaryItems(items []SummaryItem) []SummaryItem {
	if items == nil {
		return nil
	}
	out := make([]SummaryItem, 0, len(items))
	for _, item := range items {
		out = append(out, SummaryItem{
			DisplayName: item.DisplayName,
			Identifier:  "",
			Summary:     item.Summary,
			Evidence:    item.Evidence,
		})
	}
	return out
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
	results := runBoundedOrdered(ctx, refs, extraFetchConcurrency, func(ctx context.Context, ref string) DomainResult {
		args := buildArgs(ref)
		out, err := c.runner.Run(ctx, args)
		return DomainResult{
			Domain:  domain,
			Command: strings.Join(args, " "),
			Stdout:  out.Stdout,
			Stderr:  out.Stderr,
			Error:   errorString(err),
		}
	})
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

func sortedKeys(items map[string]struct{}) []string {
	out := make([]string, 0, len(items))
	for item := range items {
		out = append(out, item)
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
