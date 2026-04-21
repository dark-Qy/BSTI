package persona

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Definition struct {
	Shorthand            string `json:"shorthand"`
	ChineseLabel         string `json:"chinese_label"`
	ByteStyleDimension   string `json:"byte_style_dimension"`
	AnalysisDimension    string `json:"analysis_dimension"`
	OneLiner             string `json:"one_liner"`
	CanonicalDescription string `json:"canonical_description"`
	ImageFile            string `json:"image_file"`
}

type Primary struct {
	Shorthand            string `json:"shorthand"`
	ChineseLabel         string `json:"chinese_label"`
	ImageURL             string `json:"image_url"`
	ByteStyleDimension   string `json:"byte_style_dimension"`
	AnalysisDimension    string `json:"analysis_dimension"`
	OneLiner             string `json:"one_liner"`
	CanonicalDescription string `json:"canonical_description"`
}

type Analysis struct {
	Summary            string         `json:"summary"`
	Evidence           []EvidenceItem `json:"evidence"`
	CommunicationStyle string         `json:"communication_style"`
	WorkPreferences    string         `json:"work_preferences"`
	BlindSpots         string         `json:"blind_spots"`
	Confidence         float64        `json:"confidence"`
	Disclaimer         string         `json:"disclaimer"`
}

type EvidenceItem struct {
	Domains       []string `json:"domains"`
	Behavior      string   `json:"behavior"`
	Strength      string   `json:"strength"`
	IsCrossDomain bool     `json:"is_cross_domain"`
	IsDistinctive bool     `json:"is_distinctive"`
}

type WorkProfile struct {
	ResponsibilityScope   string   `json:"responsibility_scope"`
	TypicalWorkflow       string   `json:"typical_workflow"`
	DocWritingStyle       string   `json:"doc_writing_style"`
	DecisionMakingPattern string   `json:"decision_making_pattern"`
	TechStackOrDomain     []string `json:"tech_stack_or_domain"`
}

type parsedEvidenceItem struct {
	Domains       flexibleTextList `json:"domains"`
	Behavior      flexibleText     `json:"behavior"`
	Strength      flexibleText     `json:"strength"`
	IsCrossDomain bool             `json:"is_cross_domain"`
	IsDistinctive bool             `json:"is_distinctive"`
}

type parsedWorkProfile struct {
	ResponsibilityScope   flexibleText     `json:"responsibility_scope"`
	TypicalWorkflow       flexibleText     `json:"typical_workflow"`
	DocWritingStyle       flexibleText     `json:"doc_writing_style"`
	DecisionMakingPattern flexibleText     `json:"decision_making_pattern"`
	TechStackOrDomain     flexibleTextList `json:"tech_stack_or_domain"`
}

type parsedExpressionFingerprint struct {
	Catchphrases       flexibleTextList `json:"catchphrases"`
	Jargon             flexibleTextList `json:"jargon"`
	SentencePattern    flexibleText     `json:"sentence_pattern"`
	EmojiHabit         flexibleText     `json:"emoji_habit"`
	FormalitySpectrum  flexibleText     `json:"formality_spectrum"`
	ReplySpeedPattern  flexibleText     `json:"reply_speed_pattern"`
	ConflictExpression flexibleText     `json:"conflict_expression"`
}

type ExpressionFingerprint struct {
	Catchphrases       []string `json:"catchphrases"`
	Jargon             []string `json:"jargon"`
	SentencePattern    string   `json:"sentence_pattern"`
	EmojiHabit         string   `json:"emoji_habit"`
	FormalitySpectrum  string   `json:"formality_spectrum"`
	ReplySpeedPattern  string   `json:"reply_speed_pattern"`
	ConflictExpression string   `json:"conflict_expression"`
}

type OutputStyle struct {
	DocStructurePreference string `json:"doc_structure_preference"`
	DetailLevel            string `json:"detail_level"`
	EmailReplyPattern      string `json:"email_reply_pattern"`
	ChatReplyPattern       string `json:"chat_reply_pattern"`
	MeetingBehavior        string `json:"meeting_behavior"`
}

type parsedOutputStyle struct {
	DocStructurePreference flexibleText `json:"doc_structure_preference"`
	DetailLevel            flexibleText `json:"detail_level"`
	EmailReplyPattern      flexibleText `json:"email_reply_pattern"`
	ChatReplyPattern       flexibleText `json:"chat_reply_pattern"`
	MeetingBehavior        flexibleText `json:"meeting_behavior"`
}

type KnowledgeSignals struct {
	ExplicitOpinions []string `json:"explicit_opinions"`
	LearnedLessons   []string `json:"learned_lessons"`
	RepeatedConcerns []string `json:"repeated_concerns"`
	ReferenceSources []string `json:"reference_sources"`
}

type parsedKnowledgeSignals struct {
	ExplicitOpinions flexibleTextList `json:"explicit_opinions"`
	LearnedLessons   flexibleTextList `json:"learned_lessons"`
	RepeatedConcerns flexibleTextList `json:"repeated_concerns"`
	ReferenceSources flexibleTextList `json:"reference_sources"`
}

type BehaviorVector struct {
	Label     string `json:"label"`
	LeftPole  string `json:"left_pole"`
	RightPole string `json:"right_pole"`
	Score     int    `json:"score"`
	Summary   string `json:"summary"`
}

type InteractionTarget struct {
	DisplayName string `json:"display_name"`
	Identifier  string `json:"identifier,omitempty"`
	Summary     string `json:"summary"`
	Evidence    string `json:"evidence"`
}

type InteractionInsights struct {
	RelationshipSummary string              `json:"relationship_summary"`
	CoreCollaborators   []InteractionTarget `json:"core_collaborators"`
	FrequentPeople      []InteractionTarget `json:"frequent_people"`
	FrequentChats       []InteractionTarget `json:"frequent_chats"`
}

type ShareCard struct {
	Title           string `json:"title"`
	Subtitle        string `json:"subtitle"`
	ImageURL        string `json:"image_url"`
	DisclaimerShort string `json:"disclaimer_short"`
}

type Coverage struct {
	SuccessfulDomains []string `json:"successful_domains"`
	FailedDomains     []string `json:"failed_domains"`
	Summary           string   `json:"summary"`
}

type parsedBehaviorVector struct {
	Label     flexibleText `json:"label"`
	LeftPole  flexibleText `json:"left_pole"`
	RightPole flexibleText `json:"right_pole"`
	Score     int          `json:"score"`
	Summary   flexibleText `json:"summary"`
}

type Result struct {
	PrimaryPersona        Primary               `json:"primary_persona"`
	Analysis              Analysis              `json:"analysis"`
	WorkProfile           WorkProfile           `json:"work_profile"`
	ExpressionFingerprint ExpressionFingerprint `json:"expression_fingerprint"`
	OutputStyle           OutputStyle           `json:"output_style"`
	KnowledgeSignals      KnowledgeSignals      `json:"knowledge_signals"`
	HighlightTags         []string              `json:"highlight_tags"`
	BehaviorVectors       []BehaviorVector      `json:"behavior_vectors"`
	InteractionInsights   InteractionInsights   `json:"interaction_insights"`
	ContrastSignals       []string              `json:"contrast_signals,omitempty"`
	ShareCard             ShareCard             `json:"share_card"`
	Coverage              Coverage              `json:"coverage"`
}

type llmResponse struct {
	PrimaryPersona        string                      `json:"primary_persona"`
	Summary               flexibleText                `json:"summary"`
	Evidence              []parsedEvidenceItem        `json:"evidence"`
	WorkProfile           parsedWorkProfile           `json:"work_profile"`
	ExpressionFingerprint parsedExpressionFingerprint `json:"expression_fingerprint"`
	OutputStyle           parsedOutputStyle           `json:"output_style"`
	KnowledgeSignals      parsedKnowledgeSignals      `json:"knowledge_signals"`
	CommunicationStyle    flexibleText                `json:"communication_style"`
	WorkPreferences       flexibleText                `json:"work_preferences"`
	BlindSpots            flexibleText                `json:"blind_spots"`
	HighlightTags         flexibleTextList            `json:"highlight_tags"`
	BehaviorVectors       []parsedBehaviorVector      `json:"behavior_vectors"`
	InteractionInsights   struct {
		RelationshipSummary flexibleText              `json:"relationship_summary"`
		CoreCollaborators   []parsedInteractionTarget `json:"core_collaborators"`
		FrequentPeople      []parsedInteractionTarget `json:"frequent_people"`
		FrequentChats       []parsedInteractionTarget `json:"frequent_chats"`
	} `json:"interaction_insights"`
	ContrastSignals flexibleTextList `json:"contrast_signals"`
	Confidence      any              `json:"confidence"`
	Disclaimer      flexibleText     `json:"disclaimer"`
}

type flexibleEvidence string
type flexibleText string
type flexibleTextList []string

type parsedInteractionTarget struct {
	DisplayName flexibleText     `json:"display_name"`
	Identifier  flexibleText     `json:"identifier"`
	Summary     flexibleText     `json:"summary"`
	Evidence    flexibleEvidence `json:"evidence"`
}

func (f *flexibleText) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*f = flexibleText(strings.TrimSpace(single))
		return nil
	}

	var items []string
	if err := json.Unmarshal(data, &items); err == nil {
		parts := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item != "" {
				parts = append(parts, item)
			}
		}
		*f = flexibleText(strings.Join(parts, " / "))
		return nil
	}

	return fmt.Errorf("must be a string or string array")
}

func (f *flexibleTextList) UnmarshalJSON(data []byte) error {
	var items []string
	if err := json.Unmarshal(data, &items); err == nil {
		out := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		*f = flexibleTextList(out)
		return nil
	}

	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		single = strings.TrimSpace(single)
		if single == "" {
			*f = nil
			return nil
		}
		*f = flexibleTextList([]string{single})
		return nil
	}

	return fmt.Errorf("must be a string or string array")
}

func normalizeFlexibleText(value flexibleText) string {
	return strings.TrimSpace(string(value))
}

func normalizeFlexibleTextList(items flexibleTextList) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func normalizeParsedWorkProfile(profile parsedWorkProfile) WorkProfile {
	return WorkProfile{
		ResponsibilityScope:   normalizeFlexibleText(profile.ResponsibilityScope),
		TypicalWorkflow:       normalizeFlexibleText(profile.TypicalWorkflow),
		DocWritingStyle:       normalizeFlexibleText(profile.DocWritingStyle),
		DecisionMakingPattern: normalizeFlexibleText(profile.DecisionMakingPattern),
		TechStackOrDomain:     normalizeFlexibleTextList(profile.TechStackOrDomain),
	}
}

func normalizeParsedExpressionFingerprint(fingerprint parsedExpressionFingerprint) ExpressionFingerprint {
	return ExpressionFingerprint{
		Catchphrases:       normalizeFlexibleTextList(fingerprint.Catchphrases),
		Jargon:             normalizeFlexibleTextList(fingerprint.Jargon),
		SentencePattern:    normalizeFlexibleText(fingerprint.SentencePattern),
		EmojiHabit:         normalizeFlexibleText(fingerprint.EmojiHabit),
		FormalitySpectrum:  normalizeFlexibleText(fingerprint.FormalitySpectrum),
		ReplySpeedPattern:  normalizeFlexibleText(fingerprint.ReplySpeedPattern),
		ConflictExpression: normalizeFlexibleText(fingerprint.ConflictExpression),
	}
}

func normalizeParsedOutputStyle(style parsedOutputStyle) OutputStyle {
	return OutputStyle{
		DocStructurePreference: normalizeFlexibleText(style.DocStructurePreference),
		DetailLevel:            normalizeFlexibleText(style.DetailLevel),
		EmailReplyPattern:      normalizeFlexibleText(style.EmailReplyPattern),
		ChatReplyPattern:       normalizeFlexibleText(style.ChatReplyPattern),
		MeetingBehavior:        normalizeFlexibleText(style.MeetingBehavior),
	}
}

func normalizeParsedKnowledgeSignals(signals parsedKnowledgeSignals) KnowledgeSignals {
	return KnowledgeSignals{
		ExplicitOpinions: normalizeFlexibleTextList(signals.ExplicitOpinions),
		LearnedLessons:   normalizeFlexibleTextList(signals.LearnedLessons),
		RepeatedConcerns: normalizeFlexibleTextList(signals.RepeatedConcerns),
		ReferenceSources: normalizeFlexibleTextList(signals.ReferenceSources),
	}
}

func (e *flexibleEvidence) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*e = flexibleEvidence(strings.TrimSpace(text))
		return nil
	}

	var items []string
	if err := json.Unmarshal(data, &items); err == nil {
		parts := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item != "" {
				parts = append(parts, item)
			}
		}
		*e = flexibleEvidence(strings.Join(parts, " / "))
		return nil
	}

	return fmt.Errorf("must be a string or string array")
}

var expectedBehaviorVectors = []struct {
	Label     string
	LeftPole  string
	RightPole string
}{
	{Label: "协作方式", LeftPole: "独立成局", RightPole: "高频协同"},
	{Label: "表达风格", LeftPole: "克制压缩", RightPole: "高频输出"},
	{Label: "决策路径", LeftPole: "证据校准", RightPole: "直觉快判"},
	{Label: "推进节奏", LeftPole: "稳态推进", RightPole: "高压突进"},
	{Label: "信息处理", LeftPole: "深度聚焦", RightPole: "广度扫描"},
	{Label: "风险态度", LeftPole: "防御优先", RightPole: "进攻优先"},
}

var sensitiveIdentifierPattern = regexp.MustCompile(`\b(?:ou|oc)_[A-Za-z0-9_]+\b`)

var directMessageSignalPattern = regexp.MustCompile(`(?i)(单聊|私聊|p2p|direct message|一对一)`)

var catalog = []Definition{
	{
		Shorthand:            "BLAZE",
		ChineseLabel:         "纵火者",
		ByteStyleDimension:   "始终创业",
		AnalysisDimension:    "创业驱动力",
		OneLiner:             "每天都是 Day 1，因为我拒绝承认 Day 2",
		CanonicalDescription: "BLAZE 是从零到一的开创者，会本能地把空白视为机会，把稳定视为熵增。典型特征是持续发起新方向、对守成过敏、总想把组织重新拉回创业状态。",
		ImageFile:            "BLAZE.png",
	},
	{
		Shorthand:            "AGILE",
		ChineseLabel:         "狂奔者",
		ByteStyleDimension:   "始终创业",
		AnalysisDimension:    "敏捷执行力",
		OneLiner:             "先上线再说，hotfix 解决一切",
		CanonicalDescription: "AGILE 以极强的执行冲动和流程压缩能力著称，倾向于先做 MVP 再迭代，而不是先把讨论拉满。典型特征是高速度、低容忍低效流程、用发版代替争论。",
		ImageFile:            "AGILE.png",
	},
	{
		Shorthand:            "KEEN",
		ChineseLabel:         "雷达怪",
		ByteStyleDimension:   "始终创业",
		AnalysisDimension:    "外部敏锐度",
		OneLiner:             "你以为的护城河只是别人没修好的路",
		CanonicalDescription: "KEEN 擅长持续扫描外部世界，对用户、市场、竞品和行业变化异常敏感。典型特征是信息雷达常开、外部输入丰富，但也容易陷入收集过多、行动偏慢。",
		ImageFile:            "KEEN.png",
	},
	{
		Shorthand:            "PRISM",
		ChineseLabel:         "变色龙",
		ByteStyleDimension:   "多元兼容",
		AnalysisDimension:    "多元文化适应力",
		OneLiner:             "多线程文化模拟器，每个频道都是真的",
		CanonicalDescription: "PRISM 能在不同文化和协作语境中快速切换视角，不只是表面适配，而是真的进入对方频道理解问题。典型特征是跨文化沟通流畅、善于翻译误解，但也可能偶尔失去自己的固定坐标。",
		ImageFile:            "PRISM.png",
	},
	{
		Shorthand:            "BOND",
		ChineseLabel:         "人肉担保",
		ByteStyleDimension:   "多元兼容",
		AnalysisDimension:    "信任开放度",
		OneLiner:             "默认信任值 100，消耗光了不可充值",
		CanonicalDescription: "BOND 以高初始信任和低协作摩擦为特征，倾向于先给空间、先给信用，再看结果。典型特征是降低团队防御成本、促成合作，但对系统性辜负信任极其敏感。",
		ImageFile:            "BOND.png",
	},
	{
		Shorthand:            "FRANK",
		ChineseLabel:         "嘴替",
		ByteStyleDimension:   "坦诚清晰",
		AnalysisDimension:    "坦诚表达力",
		OneLiner:             "全公司的嘴替，大象的指认者",
		CanonicalDescription: "FRANK 的核心是把真话说出来，优先减少信息失真，而不是优先降低当下尴尬。典型特征是直面问题、拒绝粉饰，但需要不断升级表达协议，确保真话能被听进去。",
		ImageFile:            "FRANK.png",
	},
	{
		Shorthand:            "CLEAR",
		ChineseLabel:         "人形提炼机",
		ByteStyleDimension:   "坦诚清晰",
		AnalysisDimension:    "表达精准度",
		OneLiner:             "你的 2000 字邮件可以压缩成 3 句话",
		CanonicalDescription: "CLEAR 对信息密度和语言精准度要求极高，擅长把混乱讨论压缩成可执行结论。典型特征是高信噪比表达、逻辑清楚，但可能被情感型协作者误读为过冷。",
		ImageFile:            "CLEAR.png",
	},
	{
		Shorthand:            "CALM",
		ChineseLabel:         "灭火器",
		ByteStyleDimension:   "坦诚清晰",
		AnalysisDimension:    "理性沟通力",
		OneLiner:             "群里快炸了？让我来加一块冰",
		CanonicalDescription: "CALM 擅长在争议和压力中保持议题中心，把情绪从冲突中心移开，再回到事实和逻辑。典型特征是降温、稳场、就事论事，但有时会低估情绪本身也是重要信号。",
		ImageFile:            "CALM.png",
	},
	{
		Shorthand:            "CORE",
		ChineseLabel:         "刨坟者",
		ByteStyleDimension:   "求真务实",
		AnalysisDimension:    "本质洞察力",
		OneLiner:             "while(true) { 为什么？ }",
		CanonicalDescription: "CORE 会本能地追根究底，直到找到真正的底层逻辑，而不是停留在表层解释。典型特征是根因分析强、抽象能力高，但也容易在想透之前迟迟不肯推进执行。",
		ImageFile:            "CORE.png",
	},
	{
		Shorthand:            "DATA",
		ChineseLabel:         "数据教徒",
		ByteStyleDimension:   "求真务实",
		AnalysisDimension:    "实证驱动力",
		OneLiner:             "你的感觉有 p-value 吗？",
		CanonicalDescription: "DATA 对证据和一手数据高度依赖，习惯用事实而不是感觉做判断。典型特征是重视实验、原始数据和亲测，但在数据不完备的场景里可能过度保守。",
		ImageFile:            "DATA.png",
	},
	{
		Shorthand:            "REAL",
		ChineseLabel:         "验尸官",
		ByteStyleDimension:   "求真务实",
		AnalysisDimension:    "实效导向力",
		OneLiner:             "核心指标没动 = 什么都没发生",
		CanonicalDescription: "REAL 的判断标准极其务实，核心只看结果是否真实发生，而不是过程是否热闹。典型特征是终局倒推、反自嗨、强效果意识，但可能低估长期投入的慢变量价值。",
		ImageFile:            "REAL.png",
	},
	{
		Shorthand:            "DARE",
		ChineseLabel:         "赌徒",
		ByteStyleDimension:   "敢为极致",
		AnalysisDimension:    "冒险决断力",
		OneLiner:             "最大的风险是什么都不做",
		CanonicalDescription: "DARE 愿意在不确定中做非共识判断，尤其偏好下行有限、上行巨大的机会。典型特征是敢下注、敢拍板，但需要外部校准，避免把明智冒险滑成鲁莽赌博。",
		ImageFile:            "DARE.png",
	},
	{
		Shorthand:            "APEX",
		ChineseLabel:         "偏执狂",
		ByteStyleDimension:   "敢为极致",
		AnalysisDimension:    "极致追求度",
		OneLiner:             "够好了 在我这里是脏话",
		CanonicalDescription: "APEX 的内在品质标准常年高于外部要求，天然拒绝差不多就行。典型特征是关注细节、追求极致、标准稳定偏高，但也容易把团队拖入无限优化。",
		ImageFile:            "APEX.png",
	},
	{
		Shorthand:            "SCOUT",
		ChineseLabel:         "框架逃逸者",
		ByteStyleDimension:   "敢为极致",
		AnalysisDimension:    "解空间探索力",
		OneLiner:             "第一个方案永远不是最好的",
		CanonicalDescription: "SCOUT 习惯跳出当前问题的默认框架，主动探索更广的解空间。典型特征是善于重定义问题、寻找 Plan D，但也可能因为可能性太多而迟迟不收敛。",
		ImageFile:            "SCOUT.png",
	},
	{
		Shorthand:            "ROI",
		ChineseLabel:         "人形计算器",
		ByteStyleDimension:   "敢为极致",
		AnalysisDimension:    "ROI 思维力",
		OneLiner:             "这场会的人均结论产出率是多少？",
		CanonicalDescription: "ROI 会自然地把资源、时间和机会成本放进同一个式子里比较，追求整体投入产出最优。典型特征是资源配置理性、边际收益敏感，但可能过度量化不该硬算的价值。",
		ImageFile:            "ROI.png",
	},
	{
		Shorthand:            "GRIT",
		ChineseLabel:         "打不死的",
		ByteStyleDimension:   "共同成长",
		AnalysisDimension:    "韧性持久力",
		OneLiner:             "撞墙期来了？然后呢？继续跑。",
		CanonicalDescription: "GRIT 在困难和波动里依然能稳定向前，把焦虑优先转化成行动而不是停摆。典型特征是抗压、持久、能熬，但也要防止把坚持误用在已经该掉头的方向上。",
		ImageFile:            "GRIT.png",
	},
	{
		Shorthand:            "GROW",
		ChineseLabel:         "永远在学的",
		ByteStyleDimension:   "共同成长",
		AnalysisDimension:    "学习成长力",
		OneLiner:             "我是一个永远的 Beta 版本",
		CanonicalDescription: "GROW 把自己视为长期迭代中的产品，对新知识和新能力有持续内驱。典型特征是学习意愿强、边界扩展快，但也容易在输入过多时延迟产出。",
		ImageFile:            "GROW.png",
	},
	{
		Shorthand:            "NORTH",
		ChineseLabel:         "布道者",
		ByteStyleDimension:   "共同成长",
		AnalysisDimension:    "使命牵引力",
		OneLiner:             "5 秒内链接到公司愿景，不是背的",
		CanonicalDescription: "NORTH 会把团队使命和业务方向内化成自己的决策坐标，在执行泥潭里也能提醒大家为什么出发。典型特征是方向感强、能凝聚共识，但有时会忽略使命边缘的新机会。",
		ImageFile:            "NORTH.png",
	},
	{
		Shorthand:            "SPARK",
		ChineseLabel:         "脑洞制造机",
		ByteStyleDimension:   "跨维度融合",
		AnalysisDimension:    "创新突破力",
		OneLiner:             "把棋盘翻过来就是我的开局",
		CanonicalDescription: "SPARK 的优势在于远距离联想和非连续性创新，擅长把不相关的知识域碰撞成新想法。典型特征是脑洞密度高、创意跃迁快，但需要外部筛选真正能落地的点子。",
		ImageFile:            "SPARK.png",
	},
	{
		Shorthand:            "HUMBLE",
		ChineseLabel:         "自我审计员",
		ByteStyleDimension:   "跨维度融合",
		AnalysisDimension:    "谦逊自省度",
		OneLiner:             "成功时启动复盘，庆功时检查偏差",
		CanonicalDescription: "HUMBLE 擅长在成功时保持校准，不轻易把好运误判成绝对实力。典型特征是持续复盘、自我审计、反傲慢，但也要小心自省过度变成自我怀疑。",
		ImageFile:            "HUMBLE.png",
	},
}

func All() []Definition {
	out := make([]Definition, len(catalog))
	copy(out, catalog)
	return out
}

func Lookup(shorthand string) (Definition, bool) {
	key := strings.ToUpper(strings.TrimSpace(shorthand))
	for _, item := range catalog {
		if item.Shorthand == key {
			return item, true
		}
	}
	return Definition{}, false
}

func ImageURL(shorthand string) string {
	return "/assets/photos/" + strings.ToUpper(strings.TrimSpace(shorthand)) + ".png"
}

func CompactPromptCatalog(definitions []Definition) string {
	var b strings.Builder
	for _, item := range definitions {
		fmt.Fprintf(&b, "- %s | %s | %s | %s | %s | %s\n",
			item.Shorthand,
			item.ChineseLabel,
			item.ByteStyleDimension,
			item.AnalysisDimension,
			item.OneLiner,
			item.CanonicalDescription,
		)
	}
	return b.String()
}

func ParseLLMResult(raw string) (Result, error) {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 3 {
			trimmed = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var parsed llmResponse
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return Result{}, fmt.Errorf("invalid JSON output: %w", err)
	}
	if strings.TrimSpace(parsed.PrimaryPersona) == "" {
		return Result{}, fmt.Errorf("primary_persona is required")
	}
	def, ok := Lookup(parsed.PrimaryPersona)
	if !ok {
		return Result{}, fmt.Errorf("primary_persona %q is not in the BSPI catalog", parsed.PrimaryPersona)
	}
	if normalizeFlexibleText(parsed.Summary) == "" {
		return Result{}, fmt.Errorf("summary is required")
	}
	if len(parsed.Evidence) == 0 {
		return Result{}, fmt.Errorf("evidence is required")
	}
	workProfile := normalizeParsedWorkProfile(parsed.WorkProfile)
	if err := validateWorkProfile(workProfile); err != nil {
		return Result{}, err
	}
	expressionFingerprint := normalizeParsedExpressionFingerprint(parsed.ExpressionFingerprint)
	if err := validateExpressionFingerprint(expressionFingerprint); err != nil {
		return Result{}, err
	}
	outputStyle := normalizeParsedOutputStyle(parsed.OutputStyle)
	if err := validateOutputStyle(outputStyle); err != nil {
		return Result{}, err
	}
	knowledgeSignals := normalizeParsedKnowledgeSignals(parsed.KnowledgeSignals)
	if err := validateKnowledgeSignals(knowledgeSignals); err != nil {
		return Result{}, err
	}
	if normalizeFlexibleText(parsed.CommunicationStyle) == "" {
		return Result{}, fmt.Errorf("communication_style is required")
	}
	if normalizeFlexibleText(parsed.WorkPreferences) == "" {
		return Result{}, fmt.Errorf("work_preferences is required")
	}
	if normalizeFlexibleText(parsed.BlindSpots) == "" {
		return Result{}, fmt.Errorf("blind_spots is required")
	}
	confidence, err := parseConfidence(parsed.Confidence)
	if err != nil {
		return Result{}, err
	}
	normalizedHighlightTags := normalizeFlexibleTextList(parsed.HighlightTags)
	if len(normalizedHighlightTags) < 2 || len(normalizedHighlightTags) > 4 {
		return Result{}, fmt.Errorf("highlight_tags must contain 2 to 4 items")
	}
	if len(parsed.BehaviorVectors) != len(expectedBehaviorVectors) {
		return Result{}, fmt.Errorf("behavior_vectors must contain %d items", len(expectedBehaviorVectors))
	}
	if normalizeFlexibleText(parsed.InteractionInsights.RelationshipSummary) == "" {
		return Result{}, fmt.Errorf("interaction_insights.relationship_summary is required")
	}
	if normalizeFlexibleText(parsed.Disclaimer) == "" {
		return Result{}, fmt.Errorf("disclaimer is required")
	}

	evidence, err := normalizeEvidenceItems(parsed.Evidence)
	if err != nil {
		return Result{}, err
	}

	tags := normalizedHighlightTags
	if len(tags) < 2 || len(tags) > 4 {
		return Result{}, fmt.Errorf("highlight_tags must contain 2 to 4 non-empty items")
	}

	vectors := make([]BehaviorVector, 0, len(parsed.BehaviorVectors))
	for i, item := range parsed.BehaviorVectors {
		expected := expectedBehaviorVectors[i]
		if normalizeFlexibleText(item.Label) != expected.Label {
			return Result{}, fmt.Errorf("behavior_vectors[%d].label must be %q", i, expected.Label)
		}
		if normalizeFlexibleText(item.LeftPole) != expected.LeftPole {
			return Result{}, fmt.Errorf("behavior_vectors[%d].left_pole must be %q", i, expected.LeftPole)
		}
		if normalizeFlexibleText(item.RightPole) != expected.RightPole {
			return Result{}, fmt.Errorf("behavior_vectors[%d].right_pole must be %q", i, expected.RightPole)
		}
		if item.Score < 0 || item.Score > 100 {
			return Result{}, fmt.Errorf("behavior_vectors[%d].score must be between 0 and 100", i)
		}
		if normalizeFlexibleText(item.Summary) == "" {
			return Result{}, fmt.Errorf("behavior_vectors[%d].summary is required", i)
		}
		vectors = append(vectors, BehaviorVector{
			Label:     expected.Label,
			LeftPole:  expected.LeftPole,
			RightPole: expected.RightPole,
			Score:     item.Score,
			Summary:   normalizeFlexibleText(item.Summary),
		})
	}

	normalizeTargets := func(items []parsedInteractionTarget, field string) ([]InteractionTarget, error) {
		out := make([]InteractionTarget, 0, len(items))
		for i, item := range items {
			displayName := normalizeFlexibleText(item.DisplayName)
			summary := normalizeFlexibleText(item.Summary)
			evidence := strings.TrimSpace(string(item.Evidence))
			if displayName == "" {
				return nil, fmt.Errorf("interaction_insights.%s[%d].display_name is required", field, i)
			}
			if summary == "" {
				return nil, fmt.Errorf("interaction_insights.%s[%d].summary is required", field, i)
			}
			if evidence == "" {
				return nil, fmt.Errorf("interaction_insights.%s[%d].evidence is required", field, i)
			}
			target := InteractionTarget{
				DisplayName: displayName,
				Identifier:  normalizeFlexibleText(item.Identifier),
				Summary:     summary,
				Evidence:    evidence,
			}
			out = append(out, target)
		}
		return out, nil
	}

	coreCollaborators, err := normalizeTargets(parsed.InteractionInsights.CoreCollaborators, "core_collaborators")
	if err != nil {
		return Result{}, err
	}
	frequentPeople, err := normalizeTargets(parsed.InteractionInsights.FrequentPeople, "frequent_people")
	if err != nil {
		return Result{}, err
	}
	frequentChats, err := normalizeTargets(parsed.InteractionInsights.FrequentChats, "frequent_chats")
	if err != nil {
		return Result{}, err
	}
	coreCollaborators, frequentPeople, frequentChats = repairInteractionInsights(coreCollaborators, frequentPeople, frequentChats)
	result := Result{
		PrimaryPersona: Primary{
			Shorthand:            def.Shorthand,
			ChineseLabel:         def.ChineseLabel,
			ImageURL:             ImageURL(def.Shorthand),
			ByteStyleDimension:   def.ByteStyleDimension,
			AnalysisDimension:    def.AnalysisDimension,
			OneLiner:             def.OneLiner,
			CanonicalDescription: def.CanonicalDescription,
		},
		Analysis: Analysis{
			Summary:            normalizeFlexibleText(parsed.Summary),
			Evidence:           evidence,
			CommunicationStyle: normalizeFlexibleText(parsed.CommunicationStyle),
			WorkPreferences:    normalizeFlexibleText(parsed.WorkPreferences),
			BlindSpots:         normalizeFlexibleText(parsed.BlindSpots),
			Confidence:         confidence,
			Disclaimer:         normalizeFlexibleText(parsed.Disclaimer),
		},
		WorkProfile:           sanitizeWorkProfile(workProfile),
		ExpressionFingerprint: sanitizeExpressionFingerprint(expressionFingerprint),
		OutputStyle:           sanitizeOutputStyle(outputStyle),
		KnowledgeSignals:      sanitizeKnowledgeSignals(knowledgeSignals),
		HighlightTags:         tags,
		BehaviorVectors:       vectors,
		InteractionInsights: InteractionInsights{
			RelationshipSummary: normalizeFlexibleText(parsed.InteractionInsights.RelationshipSummary),
			CoreCollaborators:   coreCollaborators,
			FrequentPeople:      frequentPeople,
			FrequentChats:       frequentChats,
		},
		ContrastSignals: sanitizeStringSlice(normalizeFlexibleTextList(parsed.ContrastSignals)),
	}
	result.ShareCard = ShareCard{
		Title:           result.PrimaryPersona.ChineseLabel + " / " + result.PrimaryPersona.Shorthand,
		Subtitle:        result.PrimaryPersona.OneLiner,
		ImageURL:        result.PrimaryPersona.ImageURL,
		DisclaimerShort: result.Analysis.Disclaimer,
	}
	return SanitizeResult(result), nil
}

func SanitizeResult(result Result) Result {
	result.Analysis.Summary = scrubSensitiveIdentifiers(result.Analysis.Summary)
	result.Analysis.CommunicationStyle = scrubSensitiveIdentifiers(result.Analysis.CommunicationStyle)
	result.Analysis.WorkPreferences = scrubSensitiveIdentifiers(result.Analysis.WorkPreferences)
	result.Analysis.BlindSpots = scrubSensitiveIdentifiers(result.Analysis.BlindSpots)
	result.Analysis.Disclaimer = scrubSensitiveIdentifiers(result.Analysis.Disclaimer)
	result.Analysis.Evidence = sanitizeEvidenceItems(result.Analysis.Evidence)
	result.WorkProfile = sanitizeWorkProfile(result.WorkProfile)
	result.ExpressionFingerprint = sanitizeExpressionFingerprint(result.ExpressionFingerprint)
	result.OutputStyle = sanitizeOutputStyle(result.OutputStyle)
	result.KnowledgeSignals = sanitizeKnowledgeSignals(result.KnowledgeSignals)
	result.HighlightTags = sanitizeStringSlice(result.HighlightTags)
	result.InteractionInsights.RelationshipSummary = scrubSensitiveIdentifiers(result.InteractionInsights.RelationshipSummary)
	result.InteractionInsights.CoreCollaborators = sanitizeInteractionTargets(result.InteractionInsights.CoreCollaborators)
	result.InteractionInsights.FrequentPeople = sanitizeInteractionTargets(result.InteractionInsights.FrequentPeople)
	result.InteractionInsights.FrequentChats = sanitizeInteractionTargets(result.InteractionInsights.FrequentChats)
	result.ContrastSignals = sanitizeStringSlice(result.ContrastSignals)
	result.ShareCard.Title = scrubSensitiveIdentifiers(result.ShareCard.Title)
	result.ShareCard.Subtitle = scrubSensitiveIdentifiers(result.ShareCard.Subtitle)
	result.ShareCard.DisclaimerShort = scrubSensitiveIdentifiers(result.ShareCard.DisclaimerShort)
	result.Coverage.Summary = scrubSensitiveIdentifiers(result.Coverage.Summary)
	return result
}

func normalizeEvidenceItems(items []parsedEvidenceItem) ([]EvidenceItem, error) {
	out := make([]EvidenceItem, 0, len(items))
	for i, item := range items {
		domains := sanitizeStringSlice(normalizeFlexibleTextList(item.Domains))
		if len(domains) == 0 {
			return nil, fmt.Errorf("evidence[%d].domains is required", i)
		}
		behavior := normalizeFlexibleText(item.Behavior)
		if behavior == "" {
			return nil, fmt.Errorf("evidence[%d].behavior is required", i)
		}
		strength := normalizeFlexibleText(item.Strength)
		if strength == "" {
			return nil, fmt.Errorf("evidence[%d].strength is required", i)
		}
		out = append(out, EvidenceItem{
			Domains:       domains,
			Behavior:      behavior,
			Strength:      strength,
			IsCrossDomain: item.IsCrossDomain,
			IsDistinctive: item.IsDistinctive,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("evidence is required")
	}
	return out, nil
}

func sanitizeEvidenceItems(items []EvidenceItem) []EvidenceItem {
	if items == nil {
		return nil
	}
	out := make([]EvidenceItem, 0, len(items))
	for _, item := range items {
		out = append(out, EvidenceItem{
			Domains:       sanitizeStringSlice(item.Domains),
			Behavior:      scrubSensitiveIdentifiers(item.Behavior),
			Strength:      scrubSensitiveIdentifiers(item.Strength),
			IsCrossDomain: item.IsCrossDomain,
			IsDistinctive: item.IsDistinctive,
		})
	}
	return out
}

func sanitizeStringSlice(items []string) []string {
	if items == nil {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, scrubSensitiveIdentifiers(item))
	}
	return out
}

func sanitizeWorkProfile(profile WorkProfile) WorkProfile {
	return WorkProfile{
		ResponsibilityScope:   scrubSensitiveIdentifiers(profile.ResponsibilityScope),
		TypicalWorkflow:       scrubSensitiveIdentifiers(profile.TypicalWorkflow),
		DocWritingStyle:       scrubSensitiveIdentifiers(profile.DocWritingStyle),
		DecisionMakingPattern: scrubSensitiveIdentifiers(profile.DecisionMakingPattern),
		TechStackOrDomain:     sanitizeStringSlice(profile.TechStackOrDomain),
	}
}

func sanitizeExpressionFingerprint(fingerprint ExpressionFingerprint) ExpressionFingerprint {
	return ExpressionFingerprint{
		Catchphrases:       sanitizeStringSlice(fingerprint.Catchphrases),
		Jargon:             sanitizeStringSlice(fingerprint.Jargon),
		SentencePattern:    scrubSensitiveIdentifiers(fingerprint.SentencePattern),
		EmojiHabit:         scrubSensitiveIdentifiers(fingerprint.EmojiHabit),
		FormalitySpectrum:  scrubSensitiveIdentifiers(fingerprint.FormalitySpectrum),
		ReplySpeedPattern:  scrubSensitiveIdentifiers(fingerprint.ReplySpeedPattern),
		ConflictExpression: scrubSensitiveIdentifiers(fingerprint.ConflictExpression),
	}
}

func sanitizeOutputStyle(style OutputStyle) OutputStyle {
	return OutputStyle{
		DocStructurePreference: scrubSensitiveIdentifiers(style.DocStructurePreference),
		DetailLevel:            scrubSensitiveIdentifiers(style.DetailLevel),
		EmailReplyPattern:      scrubSensitiveIdentifiers(style.EmailReplyPattern),
		ChatReplyPattern:       scrubSensitiveIdentifiers(style.ChatReplyPattern),
		MeetingBehavior:        scrubSensitiveIdentifiers(style.MeetingBehavior),
	}
}

func sanitizeKnowledgeSignals(signals KnowledgeSignals) KnowledgeSignals {
	return KnowledgeSignals{
		ExplicitOpinions: sanitizeStringSlice(signals.ExplicitOpinions),
		LearnedLessons:   sanitizeStringSlice(signals.LearnedLessons),
		RepeatedConcerns: sanitizeStringSlice(signals.RepeatedConcerns),
		ReferenceSources: sanitizeStringSlice(signals.ReferenceSources),
	}
}

func validateWorkProfile(profile WorkProfile) error {
	if strings.TrimSpace(profile.ResponsibilityScope) == "" {
		return fmt.Errorf("work_profile.responsibility_scope is required")
	}
	if strings.TrimSpace(profile.TypicalWorkflow) == "" {
		return fmt.Errorf("work_profile.typical_workflow is required")
	}
	if strings.TrimSpace(profile.DocWritingStyle) == "" {
		return fmt.Errorf("work_profile.doc_writing_style is required")
	}
	if strings.TrimSpace(profile.DecisionMakingPattern) == "" {
		return fmt.Errorf("work_profile.decision_making_pattern is required")
	}
	if len(sanitizeStringSlice(profile.TechStackOrDomain)) == 0 {
		return fmt.Errorf("work_profile.tech_stack_or_domain is required")
	}
	return nil
}

func validateExpressionFingerprint(fingerprint ExpressionFingerprint) error {
	if len(sanitizeStringSlice(fingerprint.Catchphrases)) == 0 {
		return fmt.Errorf("expression_fingerprint.catchphrases is required")
	}
	if len(sanitizeStringSlice(fingerprint.Jargon)) == 0 {
		return fmt.Errorf("expression_fingerprint.jargon is required")
	}
	if strings.TrimSpace(fingerprint.SentencePattern) == "" {
		return fmt.Errorf("expression_fingerprint.sentence_pattern is required")
	}
	if strings.TrimSpace(fingerprint.EmojiHabit) == "" {
		return fmt.Errorf("expression_fingerprint.emoji_habit is required")
	}
	if strings.TrimSpace(fingerprint.FormalitySpectrum) == "" {
		return fmt.Errorf("expression_fingerprint.formality_spectrum is required")
	}
	if strings.TrimSpace(fingerprint.ReplySpeedPattern) == "" {
		return fmt.Errorf("expression_fingerprint.reply_speed_pattern is required")
	}
	if strings.TrimSpace(fingerprint.ConflictExpression) == "" {
		return fmt.Errorf("expression_fingerprint.conflict_expression is required")
	}
	return nil
}

func validateOutputStyle(style OutputStyle) error {
	if strings.TrimSpace(style.DocStructurePreference) == "" {
		return fmt.Errorf("output_style.doc_structure_preference is required")
	}
	if strings.TrimSpace(style.DetailLevel) == "" {
		return fmt.Errorf("output_style.detail_level is required")
	}
	if strings.TrimSpace(style.EmailReplyPattern) == "" {
		return fmt.Errorf("output_style.email_reply_pattern is required")
	}
	if strings.TrimSpace(style.ChatReplyPattern) == "" {
		return fmt.Errorf("output_style.chat_reply_pattern is required")
	}
	if strings.TrimSpace(style.MeetingBehavior) == "" {
		return fmt.Errorf("output_style.meeting_behavior is required")
	}
	return nil
}

func validateKnowledgeSignals(signals KnowledgeSignals) error {
	if len(sanitizeStringSlice(signals.ExplicitOpinions)) == 0 {
		return fmt.Errorf("knowledge_signals.explicit_opinions is required")
	}
	if len(sanitizeStringSlice(signals.LearnedLessons)) == 0 {
		return fmt.Errorf("knowledge_signals.learned_lessons is required")
	}
	if len(sanitizeStringSlice(signals.RepeatedConcerns)) == 0 {
		return fmt.Errorf("knowledge_signals.repeated_concerns is required")
	}
	if len(sanitizeStringSlice(signals.ReferenceSources)) == 0 {
		return fmt.Errorf("knowledge_signals.reference_sources is required")
	}
	return nil
}

func sanitizeInteractionTargets(items []InteractionTarget) []InteractionTarget {
	if items == nil {
		return nil
	}
	out := make([]InteractionTarget, 0, len(items))
	for _, item := range items {
		out = append(out, InteractionTarget{
			DisplayName: scrubSensitiveIdentifiers(item.DisplayName),
			Identifier:  "",
			Summary:     scrubSensitiveIdentifiers(item.Summary),
			Evidence:    scrubSensitiveIdentifiers(item.Evidence),
		})
	}
	return out
}

func repairInteractionInsights(coreCollaborators, frequentPeople, frequentChats []InteractionTarget) ([]InteractionTarget, []InteractionTarget, []InteractionTarget) {
	repairedChats := make([]InteractionTarget, 0, len(frequentChats))
	for _, item := range frequentChats {
		if !hasDirectMessageSignal(item) {
			repairedChats = append(repairedChats, item)
			continue
		}
		if mergeTargetByDisplayName(&frequentPeople, item) {
			continue
		}
		if shouldPromoteToCoreCollaborator(item) {
			if mergeTargetByDisplayName(&coreCollaborators, item) {
				continue
			}
			coreCollaborators = append(coreCollaborators, item)
			continue
		}
		frequentPeople = append(frequentPeople, item)
	}
	return coreCollaborators, frequentPeople, repairedChats
}

func mergeTargetByDisplayName(items *[]InteractionTarget, incoming InteractionTarget) bool {
	displayName := strings.TrimSpace(incoming.DisplayName)
	if displayName == "" {
		return false
	}
	for i := range *items {
		if strings.EqualFold(strings.TrimSpace((*items)[i].DisplayName), displayName) {
			(*items)[i] = mergeInteractionTarget((*items)[i], incoming)
			return true
		}
	}
	return false
}

func mergeInteractionTarget(existing, incoming InteractionTarget) InteractionTarget {
	return InteractionTarget{
		DisplayName: firstNonEmpty(existing.DisplayName, incoming.DisplayName),
		Identifier:  firstNonEmpty(existing.Identifier, incoming.Identifier),
		Summary:     mergeDistinctText(existing.Summary, incoming.Summary),
		Evidence:    mergeDistinctText(existing.Evidence, incoming.Evidence),
	}
}

func shouldPromoteToCoreCollaborator(target InteractionTarget) bool {
	combined := strings.ToLower(strings.Join([]string{
		target.DisplayName,
		target.Summary,
		target.Evidence,
	}, "\n"))
	for _, marker := range []string{"核心", "搭档", "主协作", "关键协作", "方案推进"} {
		if strings.Contains(combined, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

func mergeDistinctText(existing, incoming string) string {
	parts := []string{}
	seen := map[string]struct{}{}
	for _, candidate := range []string{existing, incoming} {
		for _, fragment := range strings.Split(candidate, " / ") {
			fragment = strings.TrimSpace(fragment)
			if fragment == "" {
				continue
			}
			if _, ok := seen[fragment]; ok {
				continue
			}
			seen[fragment] = struct{}{}
			parts = append(parts, fragment)
		}
	}
	return strings.Join(parts, " / ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func scrubSensitiveIdentifiers(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.TrimSpace(sensitiveIdentifierPattern.ReplaceAllString(value, "已隐藏ID"))
}

func hasDirectMessageSignal(target InteractionTarget) bool {
	combined := strings.Join([]string{target.DisplayName, target.Summary, target.Evidence}, "\n")
	return directMessageSignalPattern.MatchString(combined)
}

func parseConfidence(raw any) (float64, error) {
	switch value := raw.(type) {
	case float64:
		if value < 0 || value > 1 {
			return 0, fmt.Errorf("confidence must be between 0 and 1")
		}
		return value, nil
	case string:
		value = strings.TrimSpace(value)
		if value == "" {
			return 0, fmt.Errorf("confidence is required")
		}
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, fmt.Errorf("confidence must be a decimal number between 0 and 1")
		}
		if parsed < 0 || parsed > 1 {
			return 0, fmt.Errorf("confidence must be between 0 and 1")
		}
		return parsed, nil
	case nil:
		return 0, fmt.Errorf("confidence is required")
	default:
		return 0, fmt.Errorf("confidence must be a decimal number between 0 and 1")
	}
}
