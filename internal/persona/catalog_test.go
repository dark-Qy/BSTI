package persona

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogContainsAllPhotoAssets(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	items := All()
	if len(items) != 20 {
		t.Fatalf("catalog size = %d, want 20", len(items))
	}
	for _, item := range items {
		if item.ImageFile == "" {
			t.Fatalf("%s missing image file", item.Shorthand)
		}
		if _, err := os.Stat(filepath.Join(root, "photos", item.ImageFile)); err != nil {
			t.Fatalf("%s missing photo %s: %v", item.Shorthand, item.ImageFile, err)
		}
	}
}

func TestParseLLMResultBuildsStructuredPersonaResult(t *testing.T) {
	raw := `{
	  "primary_persona": "PRISM",
	  "summary": "最显著的区分性行为是会主动把不同角色的话翻译成共同问题，这个信号同时出现在 chat 和 docs 中。跨域一致性体现在其在群聊中负责收敛分歧，在文档里负责把不同背景的输入整理成统一结构。关键证据来自高频跨团队沟通与持续的文档归纳动作，因此将其归为 PRISM。虽然群聊输出密度高，但在正式文档里保持克制结构化，这种场景反差并不影响其核心桥接特征。",
	  "evidence": [
	    {
	      "domains": ["chat", "docs"],
	      "behavior": "持续在不同对象之间切换表达方式并翻译语境差异",
	      "strength": "近 30 天高频出现",
	      "is_cross_domain": true,
	      "is_distinctive": true
	    },
	    {
	      "domains": ["chat", "calendar"],
	      "behavior": "跨团队讨论和会议中经常承担信息收敛角色",
	      "strength": "多次关键节点出现",
	      "is_cross_domain": true,
	      "is_distinctive": false
	    }
	  ],
	  "work_profile": {
	    "responsibility_scope": "负责跨团队协作推进和多语境问题桥接。",
	    "typical_workflow": "先收集多方输入，再整理共识，再推进后续落地。",
	    "doc_writing_style": "擅长把不同来源的输入整合成统一结构。",
	    "decision_making_pattern": "先校准上下文和角色诉求，再形成判断。",
	    "tech_stack_or_domain": ["协作流程", "跨团队沟通", "方案整理"]
	  },
	  "expression_fingerprint": {
	    "catchphrases": ["先对齐语境", "我帮大家翻一下", "先收一下结论"],
	    "jargon": ["对齐", "上下文", "桥接"],
	    "sentence_pattern": "会先复述不同角色关注点，再给出统一结论。",
	    "emoji_habit": "低频使用，偶尔用 👍 表示确认。",
	    "formality_spectrum": "在正式文档中偏正式，在群聊里更轻量。",
	    "reply_speed_pattern": "工作时间高频响应，关键议题会快速跟进。",
	    "conflict_expression": "不直接否定，而是先翻译分歧来源。"
	  },
	  "output_style": {
	    "doc_structure_preference": "偏好多层级标题与问题-结论式整理。",
	    "detail_level": "适中偏详尽",
	    "email_reply_pattern": "先总结结论，再解释不同角色影响。",
	    "chat_reply_pattern": "在关键分歧点集中输出，平时更多做短确认。",
	    "meeting_behavior": "在多人讨论中承担翻译和收敛角色。"
	  },
	  "knowledge_signals": {
	    "explicit_opinions": ["跨团队问题先对齐语境再谈方案。"],
	    "learned_lessons": ["角色认知不统一时，直接推进很容易反复返工。"],
	    "repeated_concerns": ["上下文偏差", "角色误解"],
	    "reference_sources": ["会议结论", "跨团队聊天记录", "方案文档"]
	  },
	  "communication_style": "会主动翻译语境差异，让不同角色更快对齐。",
	  "work_preferences": "偏好多方协同、跨语境问题和需要桥接认知差异的场景。",
	  "blind_spots": "容易长期适配别人，偶尔忽略自己的稳定表达方式。",
	  "highlight_tags": ["跨团队桥接", "语境切换", "高频协同"],
	  "behavior_vectors": [
	    {
	      "label": "协作方式",
	      "left_pole": "独立成局",
	      "right_pole": "高频协同",
	      "score": 82,
	      "summary": "更容易在多人协同和跨团队桥接场景里发挥影响力。"
	    },
	    {
	      "label": "表达风格",
	      "left_pole": "克制压缩",
	      "right_pole": "高频输出",
	      "score": 71,
	      "summary": "会根据对象切换表达密度，但整体偏主动输出。"
	    },
	    {
	      "label": "决策路径",
	      "left_pole": "证据校准",
	      "right_pole": "直觉快判",
	      "score": 44,
	      "summary": "更倾向先校准语境和事实，再推动形成判断。"
	    },
	    {
	      "label": "推进节奏",
	      "left_pole": "稳态推进",
	      "right_pole": "高压突进",
	      "score": 58,
	      "summary": "整体节奏稳中偏快，遇到窗口期会明显提速。"
	    },
	    {
	      "label": "信息处理",
	      "left_pole": "深度聚焦",
	      "right_pole": "广度扫描",
	      "score": 73,
	      "summary": "擅长在多个角色和上下文之间切换，先做广度整合。"
	    },
	    {
	      "label": "风险态度",
	      "left_pole": "防御优先",
	      "right_pole": "进攻优先",
	      "score": 41,
	      "summary": "会先避免语境错位带来的返工，再推进方案。"
	    }
	  ],
	  "interaction_insights": {
	    "relationship_summary": "对外沟通和跨团队桥接都很活跃，既有稳定单聊协作者，也有高频参与的项目群。",
	    "core_collaborators": [
	      {
	        "display_name": "小李",
	        "identifier": "ou_core_1",
	        "summary": "在关键问题拆解和方案推进中频繁互相校准。",
	        "evidence": "近 30 天内多次 direct message 往返，并伴随方案细节追问。"
	      }
	    ],
	    "frequent_people": [
	      {
	        "display_name": "小李",
	        "identifier": "ou_core_1",
	        "summary": "高频讨论技术细节与推进节奏。",
	        "evidence": "P2P 往返密度高，且多出现在工作日连续窗口。"
	      }
	    ],
	    "frequent_chats": [
	      {
	        "display_name": "跨团队项目群",
	        "identifier": "oc_chat_1",
	        "summary": "承担跨角色信息同步和结论收敛。",
	        "evidence": "最近一个月多次在该群发起和回应关键讨论。"
	      }
	    ]
	  },
	  "confidence": 0.86,
	  "disclaimer": "仅基于授权数据的行为风格观察，不是心理诊断。"
	}`

	got, err := ParseLLMResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.PrimaryPersona.Shorthand != "PRISM" {
		t.Fatalf("shorthand = %q", got.PrimaryPersona.Shorthand)
	}
	if got.PrimaryPersona.ImageURL != "/assets/photos/PRISM.png" {
		t.Fatalf("image_url = %q", got.PrimaryPersona.ImageURL)
	}
	if !strings.Contains(got.PrimaryPersona.CanonicalDescription, "跨文化") {
		t.Fatalf("canonical_description = %q", got.PrimaryPersona.CanonicalDescription)
	}
	if len(got.Analysis.Evidence) != 2 {
		t.Fatalf("evidence = %#v", got.Analysis.Evidence)
	}
	if got.Analysis.Evidence[0].Behavior == "" || len(got.Analysis.Evidence[0].Domains) != 2 {
		t.Fatalf("evidence[0] = %#v", got.Analysis.Evidence[0])
	}
	if got.Analysis.Confidence != 0.86 {
		t.Fatalf("confidence = %v", got.Analysis.Confidence)
	}
	if got.WorkProfile.ResponsibilityScope == "" || len(got.WorkProfile.TechStackOrDomain) != 3 {
		t.Fatalf("work_profile = %#v", got.WorkProfile)
	}
	if len(got.ExpressionFingerprint.Catchphrases) != 3 {
		t.Fatalf("expression_fingerprint = %#v", got.ExpressionFingerprint)
	}
	if got.OutputStyle.MeetingBehavior == "" {
		t.Fatalf("output_style = %#v", got.OutputStyle)
	}
	if len(got.KnowledgeSignals.RepeatedConcerns) != 2 {
		t.Fatalf("knowledge_signals = %#v", got.KnowledgeSignals)
	}
	if len(got.HighlightTags) != 3 {
		t.Fatalf("highlight_tags = %#v", got.HighlightTags)
	}
	if len(got.BehaviorVectors) != 6 {
		t.Fatalf("behavior_vectors = %#v", got.BehaviorVectors)
	}
	if got.BehaviorVectors[0].Label != "协作方式" || got.BehaviorVectors[0].Score != 82 {
		t.Fatalf("behavior_vector[0] = %#v", got.BehaviorVectors[0])
	}
	if got.InteractionInsights.RelationshipSummary == "" {
		t.Fatalf("interaction_insights = %#v", got.InteractionInsights)
	}
	if len(got.InteractionInsights.CoreCollaborators) != 1 {
		t.Fatalf("core_collaborators = %#v", got.InteractionInsights.CoreCollaborators)
	}
}

func TestParseLLMResultRejectsUnknownPersona(t *testing.T) {
	raw := validLLMJSON("UNKNOWN", `"confidence": 0.42,`)

	_, err := ParseLLMResult(raw)
	if err == nil || !strings.Contains(err.Error(), "not in the BSPI catalog") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseLLMResultAcceptsJSONCodeFence(t *testing.T) {
	raw := "```json\n" + validLLMJSON("CALM", `"confidence": 0.75,`) + "\n```"
	got, err := ParseLLMResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.PrimaryPersona.Shorthand != "CALM" {
		t.Fatalf("shorthand = %q", got.PrimaryPersona.Shorthand)
	}
}

func TestParseLLMResultAcceptsStringConfidence(t *testing.T) {
	raw := validLLMJSON("CALM", `"confidence": "0.64",`)
	got, err := ParseLLMResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Analysis.Confidence != 0.64 {
		t.Fatalf("confidence = %v", got.Analysis.Confidence)
	}
}

func TestParseLLMResultRejectsOutOfRangeConfidence(t *testing.T) {
	raw := validLLMJSON("CALM", `"confidence": 1.2,`)
	_, err := ParseLLMResult(raw)
	if err == nil || !strings.Contains(err.Error(), "confidence must be between 0 and 1") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseLLMResultRejectsMissingInteractionRelationshipSummary(t *testing.T) {
	raw := validLLMJSON("CALM", `"interaction_insights": {"relationship_summary": "", "core_collaborators": [], "frequent_people": [], "frequent_chats": []},
	  "confidence": 0.64,`)
	_, err := ParseLLMResult(raw)
	if err == nil || !strings.Contains(err.Error(), "interaction_insights.relationship_summary is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseLLMResultAcceptsInteractionEvidenceArray(t *testing.T) {
	raw := validLLMJSON("PRISM", `"interaction_insights": {
	    "relationship_summary": "x",
	    "core_collaborators": [{"display_name": "小李", "identifier": "ou_core_1", "summary": "x", "evidence": ["证据 1 ou_core_1", "证据 2"]}],
	    "frequent_people": [{"display_name": "小王", "identifier": "ou_freq_1", "summary": "x", "evidence": ["证据 A", "证据 B"]}],
	    "frequent_chats": [{"display_name": "项目群", "identifier": "oc_chat_1", "summary": "x", "evidence": ["证据甲", "证据乙"]}]
	  },
	  "confidence": 0.64,`)

	got, err := ParseLLMResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.InteractionInsights.CoreCollaborators[0].Evidence != "证据 1 已隐藏ID / 证据 2" {
		t.Fatalf("core evidence = %q", got.InteractionInsights.CoreCollaborators[0].Evidence)
	}
	if got.InteractionInsights.FrequentPeople[0].Evidence != "证据 A / 证据 B" {
		t.Fatalf("people evidence = %q", got.InteractionInsights.FrequentPeople[0].Evidence)
	}
	if got.InteractionInsights.FrequentChats[0].Evidence != "证据甲 / 证据乙" {
		t.Fatalf("chat evidence = %q", got.InteractionInsights.FrequentChats[0].Evidence)
	}
	if got.InteractionInsights.CoreCollaborators[0].Identifier != "" {
		t.Fatalf("core identifier = %q", got.InteractionInsights.CoreCollaborators[0].Identifier)
	}
	if got.InteractionInsights.FrequentPeople[0].Identifier != "" {
		t.Fatalf("people identifier = %q", got.InteractionInsights.FrequentPeople[0].Identifier)
	}
	if got.InteractionInsights.FrequentChats[0].Identifier != "" {
		t.Fatalf("chat identifier = %q", got.InteractionInsights.FrequentChats[0].Identifier)
	}
}

func TestParseLLMResultRepairsDirectMessageSignalsInFrequentChats(t *testing.T) {
	raw := validLLMJSON("PRISM", `"interaction_insights": {
	    "relationship_summary": "x",
	    "core_collaborators": [],
	    "frequent_people": [],
	    "frequent_chats": [{"display_name": "与小李单聊", "summary": "direct message 高频往返", "evidence": "最近 30 天单聊出现很多次"}]
	  },
	  "confidence": 0.64,`)

	got, err := ParseLLMResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.InteractionInsights.FrequentChats) != 0 {
		t.Fatalf("frequent_chats = %#v", got.InteractionInsights.FrequentChats)
	}
	if len(got.InteractionInsights.FrequentPeople) != 1 {
		t.Fatalf("frequent_people = %#v", got.InteractionInsights.FrequentPeople)
	}
	if got.InteractionInsights.FrequentPeople[0].DisplayName != "与小李单聊" {
		t.Fatalf("migrated target = %#v", got.InteractionInsights.FrequentPeople[0])
	}
}

func TestParseLLMResultRepairsMixedFrequentChatsAndMergesByDisplayName(t *testing.T) {
	raw := validLLMJSON("PRISM", `"interaction_insights": {
	    "relationship_summary": "x",
	    "core_collaborators": [],
	    "frequent_people": [{"display_name": "小李", "summary": "高频协作对象", "evidence": "原有人际信号"}],
	    "frequent_chats": [
	      {"display_name": "项目群", "summary": "高频同步群", "evidence": "多次出现关键结论同步"},
	      {"display_name": "小李", "summary": "direct message 高频往返", "evidence": "最近 30 天单聊出现很多次"}
	    ]
	  },
	  "confidence": 0.64,`)

	got, err := ParseLLMResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.InteractionInsights.FrequentChats) != 1 {
		t.Fatalf("frequent_chats = %#v", got.InteractionInsights.FrequentChats)
	}
	if got.InteractionInsights.FrequentChats[0].DisplayName != "项目群" {
		t.Fatalf("group chat target = %#v", got.InteractionInsights.FrequentChats[0])
	}
	if len(got.InteractionInsights.FrequentPeople) != 1 {
		t.Fatalf("frequent_people = %#v", got.InteractionInsights.FrequentPeople)
	}
	if got.InteractionInsights.FrequentPeople[0].DisplayName != "小李" {
		t.Fatalf("merged person = %#v", got.InteractionInsights.FrequentPeople[0])
	}
	if !strings.Contains(got.InteractionInsights.FrequentPeople[0].Evidence, "原有人际信号") || !strings.Contains(got.InteractionInsights.FrequentPeople[0].Evidence, "最近 30 天单聊出现很多次") {
		t.Fatalf("merged evidence = %q", got.InteractionInsights.FrequentPeople[0].Evidence)
	}
}

func TestParseLLMResultAcceptsBlindSpotsArray(t *testing.T) {
	raw := validLLMJSON("CALM", `"blind_spots": ["容易延后表态", "在低信息密度讨论里可能显得保守"],
	  "confidence": 0.64,`)

	got, err := ParseLLMResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Analysis.BlindSpots, "容易延后表态") {
		t.Fatalf("blind_spots = %q", got.Analysis.BlindSpots)
	}
	if !strings.Contains(got.Analysis.BlindSpots, "在低信息密度讨论里可能显得保守") {
		t.Fatalf("blind_spots = %q", got.Analysis.BlindSpots)
	}
}

func TestParseLLMResultAcceptsWorkProfileTextArrays(t *testing.T) {
	raw := validLLMJSON("CALM", `"work_profile": {
	    "responsibility_scope": ["负责稳定推进", "承担跨团队对齐"],
	    "typical_workflow": ["先收集输入", "再整理结论", "最后推进落地"],
	    "doc_writing_style": ["结构化", "偏结论先行"],
	    "decision_making_pattern": ["先看证据", "再形成判断"],
	    "tech_stack_or_domain": "RAG"
	  },
	  "confidence": 0.64,`)

	got, err := ParseLLMResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkProfile.TypicalWorkflow != "先收集输入 / 再整理结论 / 最后推进落地" {
		t.Fatalf("typical_workflow = %q", got.WorkProfile.TypicalWorkflow)
	}
	if got.WorkProfile.ResponsibilityScope != "负责稳定推进 / 承担跨团队对齐" {
		t.Fatalf("responsibility_scope = %q", got.WorkProfile.ResponsibilityScope)
	}
	if got.WorkProfile.DocWritingStyle != "结构化 / 偏结论先行" {
		t.Fatalf("doc_writing_style = %q", got.WorkProfile.DocWritingStyle)
	}
	if got.WorkProfile.DecisionMakingPattern != "先看证据 / 再形成判断" {
		t.Fatalf("decision_making_pattern = %q", got.WorkProfile.DecisionMakingPattern)
	}
	if len(got.WorkProfile.TechStackOrDomain) != 1 || got.WorkProfile.TechStackOrDomain[0] != "RAG" {
		t.Fatalf("tech_stack_or_domain = %#v", got.WorkProfile.TechStackOrDomain)
	}
}

func TestParseLLMResultAcceptsOtherFlexibleTextFields(t *testing.T) {
	raw := `{
	  "primary_persona": "CALM",
	  "summary": ["先稳住场面", "再推动事实对齐"],
	  "evidence": [{"domains": "chat", "behavior": ["会先收敛分歧", "再给出判断"], "strength": ["高频出现", "跨会话稳定"], "is_cross_domain": false, "is_distinctive": false}],
	  "work_profile": {
	    "responsibility_scope": "x",
	    "typical_workflow": "x",
	    "doc_writing_style": "x",
	    "decision_making_pattern": "x",
	    "tech_stack_or_domain": ["x"]
	  },
	  "expression_fingerprint": {
	    "catchphrases": "先对齐一下",
	    "jargon": "对齐",
	    "sentence_pattern": ["先复述问题", "再给结论"],
	    "emoji_habit": ["低频使用", "偶尔点头"],
	    "formality_spectrum": ["群聊偏轻量", "文档偏正式"],
	    "reply_speed_pattern": ["关键问题快", "一般问题稳"],
	    "conflict_expression": ["先讲事实", "再提分歧"]
	  },
	  "output_style": {
	    "doc_structure_preference": ["结论先行", "分点展开"],
	    "detail_level": ["适中", "必要时展开"],
	    "email_reply_pattern": ["先回结论", "再补依据"],
	    "chat_reply_pattern": ["先短答", "再补充"],
	    "meeting_behavior": ["先稳场", "再推进"]
	  },
	  "knowledge_signals": {
	    "explicit_opinions": "先看事实",
	    "learned_lessons": "避免过早下结论",
	    "repeated_concerns": "信息不完整",
	    "reference_sources": "会议纪要"
	  },
	  "communication_style": ["先对齐事实", "再讨论分歧"],
	  "work_preferences": ["偏好稳定推进", "不喜欢噪声争论"],
	  "blind_spots": "x",
	  "highlight_tags": ["冷静控场", "先稳后推"],
	  "behavior_vectors": [
	    {"label":"协作方式","left_pole":"独立成局","right_pole":"高频协同","score":60,"summary":"x"},
	    {"label":"表达风格","left_pole":"克制压缩","right_pole":"高频输出","score":40,"summary":"x"},
	    {"label":"决策路径","left_pole":"证据校准","right_pole":"直觉快判","score":35,"summary":"x"},
	    {"label":"推进节奏","left_pole":"稳态推进","right_pole":"高压突进","score":48,"summary":"x"},
	    {"label":"信息处理","left_pole":"深度聚焦","right_pole":"广度扫描","score":52,"summary":"x"},
	    {"label":"风险态度","left_pole":"防御优先","right_pole":"进攻优先","score":45,"summary":"x"}
	  ],
	  "interaction_insights": {
	    "relationship_summary": ["稳定协作较多", "偏好事实对齐"],
	    "core_collaborators": [{"display_name": ["小李"], "identifier": ["ou_core_1"], "summary": ["关键问题高频协作"], "evidence": ["多次讨论方案", "经常互相校准"]}],
	    "frequent_people": [],
	    "frequent_chats": []
	  },
	  "contrast_signals": "先稳后推",
	  "confidence": 0.64,
	  "disclaimer": ["仅基于授权数据", "不是心理诊断"]
	}`

	got, err := ParseLLMResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Analysis.Summary != "先稳住场面 / 再推动事实对齐" {
		t.Fatalf("summary = %q", got.Analysis.Summary)
	}
	if len(got.Analysis.Evidence) != 1 || got.Analysis.Evidence[0].Behavior != "会先收敛分歧 / 再给出判断" || len(got.Analysis.Evidence[0].Domains) != 1 || got.Analysis.Evidence[0].Domains[0] != "chat" {
		t.Fatalf("evidence = %#v", got.Analysis.Evidence)
	}
	if len(got.ExpressionFingerprint.Catchphrases) != 1 || got.ExpressionFingerprint.Catchphrases[0] != "先对齐一下" {
		t.Fatalf("catchphrases = %#v", got.ExpressionFingerprint.Catchphrases)
	}
	if got.ExpressionFingerprint.SentencePattern != "先复述问题 / 再给结论" {
		t.Fatalf("sentence_pattern = %q", got.ExpressionFingerprint.SentencePattern)
	}
	if got.OutputStyle.DocStructurePreference != "结论先行 / 分点展开" {
		t.Fatalf("doc_structure_preference = %q", got.OutputStyle.DocStructurePreference)
	}
	if len(got.KnowledgeSignals.ExplicitOpinions) != 1 || got.KnowledgeSignals.ExplicitOpinions[0] != "先看事实" {
		t.Fatalf("knowledge_signals = %#v", got.KnowledgeSignals)
	}
	if got.Analysis.CommunicationStyle != "先对齐事实 / 再讨论分歧" {
		t.Fatalf("communication_style = %q", got.Analysis.CommunicationStyle)
	}
	if got.Analysis.WorkPreferences != "偏好稳定推进 / 不喜欢噪声争论" {
		t.Fatalf("work_preferences = %q", got.Analysis.WorkPreferences)
	}
	if len(got.HighlightTags) != 2 || got.HighlightTags[0] != "冷静控场" || got.HighlightTags[1] != "先稳后推" {
		t.Fatalf("highlight_tags = %#v", got.HighlightTags)
	}
	if got.InteractionInsights.RelationshipSummary != "稳定协作较多 / 偏好事实对齐" {
		t.Fatalf("relationship_summary = %q", got.InteractionInsights.RelationshipSummary)
	}
	if len(got.InteractionInsights.CoreCollaborators) != 1 || got.InteractionInsights.CoreCollaborators[0].DisplayName != "小李" {
		t.Fatalf("core_collaborators = %#v", got.InteractionInsights.CoreCollaborators)
	}
	if len(got.ContrastSignals) != 1 || got.ContrastSignals[0] != "先稳后推" {
		t.Fatalf("contrast_signals = %#v", got.ContrastSignals)
	}
	if got.Analysis.Disclaimer != "仅基于授权数据 / 不是心理诊断" {
		t.Fatalf("disclaimer = %q", got.Analysis.Disclaimer)
	}
}

func validLLMJSON(personaName, tail string) string {
	return `{
	  "primary_persona": "` + personaName + `",
	  "summary": "x",
	  "evidence": [{"domains": ["chat"], "behavior": "x", "strength": "高频", "is_cross_domain": false, "is_distinctive": false}],
	  "work_profile": {
	    "responsibility_scope": "x",
	    "typical_workflow": "x",
	    "doc_writing_style": "x",
	    "decision_making_pattern": "x",
	    "tech_stack_or_domain": ["x"]
	  },
	  "expression_fingerprint": {
	    "catchphrases": ["x"],
	    "jargon": ["x"],
	    "sentence_pattern": "x",
	    "emoji_habit": "x",
	    "formality_spectrum": "x",
	    "reply_speed_pattern": "x",
	    "conflict_expression": "x"
	  },
	  "output_style": {
	    "doc_structure_preference": "x",
	    "detail_level": "x",
	    "email_reply_pattern": "x",
	    "chat_reply_pattern": "x",
	    "meeting_behavior": "x"
	  },
	  "knowledge_signals": {
	    "explicit_opinions": ["x"],
	    "learned_lessons": ["x"],
	    "repeated_concerns": ["x"],
	    "reference_sources": ["x"]
	  },
	  "communication_style": "x",
	  "work_preferences": "x",
	  "blind_spots": "x",
	  "highlight_tags": ["冷静控场", "理性沟通"],
	  "behavior_vectors": [
	    {"label":"协作方式","left_pole":"独立成局","right_pole":"高频协同","score":60,"summary":"x"},
	    {"label":"表达风格","left_pole":"克制压缩","right_pole":"高频输出","score":40,"summary":"x"},
	    {"label":"决策路径","left_pole":"证据校准","right_pole":"直觉快判","score":35,"summary":"x"},
	    {"label":"推进节奏","left_pole":"稳态推进","right_pole":"高压突进","score":48,"summary":"x"},
	    {"label":"信息处理","left_pole":"深度聚焦","right_pole":"广度扫描","score":52,"summary":"x"},
	    {"label":"风险态度","left_pole":"防御优先","right_pole":"进攻优先","score":45,"summary":"x"}
	  ],
	  "interaction_insights": {"relationship_summary": "x", "core_collaborators": [], "frequent_people": [], "frequent_chats": []},
	  ` + tail + `
	  "disclaimer": "x"
	}`
}
