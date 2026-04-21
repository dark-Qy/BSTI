package report

import (
	"os"
	"strings"
	"testing"

	"feishu-personality-agent/internal/persona"
)

func TestWriteLocalReportWritesStructuredMarkdownAndHTML(t *testing.T) {
	dir := t.TempDir()
	result := persona.Result{
		PrimaryPersona: persona.Primary{
			Shorthand:            "PRISM",
			ChineseLabel:         "变色龙",
			ImageURL:             "/assets/photos/PRISM.png",
			ByteStyleDimension:   "多元兼容",
			AnalysisDimension:    "多元文化适应力",
			OneLiner:             "多线程文化模拟器，每个频道都是真的",
			CanonicalDescription: "PRISM 擅长跨文化切换与桥接。",
		},
		Analysis: persona.Analysis{
			Summary: "跨文化语境切换自然，适合做协作桥梁。",
			Evidence: []persona.EvidenceItem{
				{Domains: []string{"chat", "docs"}, Behavior: "频繁在不同协作对象之间切换表达方式", Strength: "高频", IsCrossDomain: true, IsDistinctive: true},
				{Domains: []string{"chat", "calendar"}, Behavior: "跨团队沟通密度高", Strength: "多次关键节点出现", IsCrossDomain: true, IsDistinctive: false},
			},
			CommunicationStyle: "先理解对方语境，再翻译回共同问题。",
			WorkPreferences:    "偏好多方协作与需要桥接认知差异的任务。",
			BlindSpots:         "可能长期适配别人而忽略自己的固定表达方式。",
			Confidence:         0.86,
			Disclaimer:         "仅基于授权数据的行为风格观察。",
		},
		WorkProfile: persona.WorkProfile{
			ResponsibilityScope:   "负责跨团队协作推进。",
			TypicalWorkflow:       "先收集输入，再统一结构，再推进落实。",
			DocWritingStyle:       "偏好多级标题和统一模板。",
			DecisionMakingPattern: "先校准上下文再判断。",
			TechStackOrDomain:     []string{"协作流程", "方案整理"},
		},
		ExpressionFingerprint: persona.ExpressionFingerprint{
			Catchphrases:       []string{"先对齐一下", "我帮大家收一下"},
			Jargon:             []string{"对齐", "上下文"},
			SentencePattern:    "先复述差异，再给结论。",
			EmojiHabit:         "低频使用 👍。",
			FormalitySpectrum:  "正式文档偏正式，群聊偏轻量。",
			ReplySpeedPattern:  "关键议题快速响应。",
			ConflictExpression: "先翻译分歧来源。",
		},
		OutputStyle: persona.OutputStyle{
			DocStructurePreference: "多级标题 + 列点归纳",
			DetailLevel:            "适中偏详尽",
			EmailReplyPattern:      "先结论再展开",
			ChatReplyPattern:       "关键节点集中输出",
			MeetingBehavior:        "讨论中负责收敛",
		},
		KnowledgeSignals: persona.KnowledgeSignals{
			ExplicitOpinions: []string{"跨团队问题先对齐语境。"},
			LearnedLessons:   []string{"认知不统一时直接推进容易返工。"},
			RepeatedConcerns: []string{"上下文偏差"},
			ReferenceSources: []string{"会议结论", "方案文档"},
		},
		HighlightTags: []string{"跨团队桥接", "语境切换", "协作雷达"},
		BehaviorVectors: []persona.BehaviorVector{
			{Label: "协作方式", LeftPole: "独立成局", RightPole: "高频协同", Score: 82, Summary: "多人协同场景更强。"},
			{Label: "表达风格", LeftPole: "克制压缩", RightPole: "高频输出", Score: 71, Summary: "输出密度较高。"},
			{Label: "决策路径", LeftPole: "证据校准", RightPole: "直觉快判", Score: 44, Summary: "先校准事实再下判断。"},
			{Label: "推进节奏", LeftPole: "稳态推进", RightPole: "高压突进", Score: 58, Summary: "稳中偏快。"},
			{Label: "信息处理", LeftPole: "深度聚焦", RightPole: "广度扫描", Score: 68, Summary: "会先做多角色整合。"},
			{Label: "风险态度", LeftPole: "防御优先", RightPole: "进攻优先", Score: 39, Summary: "更关注减少误解成本。"},
		},
		InteractionInsights: persona.InteractionInsights{
			RelationshipSummary: "对外沟通和跨团队桥接都很活跃。",
			CoreCollaborators: []persona.InteractionTarget{
				{DisplayName: "小李", Identifier: "ou_core_1", Summary: "方案推进搭档", Evidence: "最近一个月 direct message 频繁来回。"},
			},
			FrequentPeople: []persona.InteractionTarget{
				{DisplayName: "小李", Identifier: "ou_core_1", Summary: "高频技术讨论对象", Evidence: "连续多周高密度互动。"},
			},
			FrequentChats: []persona.InteractionTarget{
				{DisplayName: "跨团队项目群", Identifier: "oc_chat_1", Summary: "高频同步群", Evidence: "多次在群内发起与回应结论收敛。"},
			},
		},
		Coverage: persona.Coverage{
			SuccessfulDomains: []string{"chat", "docs", "calendar"},
			FailedDomains:     []string{"mail"},
			Summary:           "已覆盖 3 个数据域，1 个数据域因权限受限未纳入。",
		},
	}

	paths, err := WriteLocal(dir, result)
	if err != nil {
		t.Fatal(err)
	}
	htmlBytes, err := os.ReadFile(paths.HTML)
	if err != nil {
		t.Fatal(err)
	}
	html := string(htmlBytes)
	for _, want := range []string{"PRISM / 变色龙", "/assets/photos/PRISM.png", "官方人格定义", "个体分析摘要", "跨文化切换与桥接", "行为信号", "工作画像", "表达指纹", "知识信号", "跨团队桥接", "互动关系", "跨团队项目群", "数据覆盖"} {
		if !strings.Contains(html, want) {
			t.Fatalf("html missing %q: %s", want, html)
		}
	}
	for _, unwanted := range []string{"ou_core_1", "oc_chat_1"} {
		if strings.Contains(html, unwanted) {
			t.Fatalf("html leaked identifier %q: %s", unwanted, html)
		}
	}
	mdBytes, err := os.ReadFile(paths.Markdown)
	if err != nil {
		t.Fatal(err)
	}
	md := string(mdBytes)
	if !strings.Contains(md, "## Primary Persona") || !strings.Contains(md, "PRISM") || !strings.Contains(md, "## Behavior Vectors") {
		t.Fatalf("markdown = %s", md)
	}
	if !strings.Contains(md, "## Work Profile") || !strings.Contains(md, "## Expression Fingerprint") || !strings.Contains(md, "## Knowledge Signals") {
		t.Fatalf("markdown upgraded sections = %s", md)
	}
	if !strings.Contains(md, "## Interaction Insights") || !strings.Contains(md, "跨团队项目群") {
		t.Fatalf("markdown interaction = %s", md)
	}
	for _, unwanted := range []string{"ou_core_1", "oc_chat_1"} {
		if strings.Contains(md, unwanted) {
			t.Fatalf("markdown leaked identifier %q: %s", unwanted, md)
		}
	}
	if !strings.Contains(md, "## 置信度\n0.86") {
		t.Fatalf("markdown confidence = %s", md)
	}
	if !strings.Contains(html, "置信度：</strong>0.86") {
		t.Fatalf("html confidence = %s", html)
	}
}
