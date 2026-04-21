package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"feishu-personality-agent/internal/collector"
	"feishu-personality-agent/internal/config"
	"feishu-personality-agent/internal/persona"
	"feishu-personality-agent/internal/session"
)

func TestHealthzReturnsOK(t *testing.T) {
	srv := New(ServerConfig{
		App:   config.Config{},
		Store: session.NewFileStore(t.TempDir()),
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("status body = %#v", resp)
	}
}

func TestServerCreatesSessionAndStartsLogin(t *testing.T) {
	store := session.NewFileStore(t.TempDir())
	srv := New(ServerConfig{
		App:   config.Config{},
		Store: store,
		LoginStarter: func(ctx context.Context, s *session.Session) (string, error) {
			s.Status = session.StatusLoginPending
			s.VerificationURL = "https://verify.example"
			return s.VerificationURL, store.Save(s)
		},
		Analyzer: func(ctx context.Context, s *session.Session) error {
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d body=%s", w.Code, w.Body.String())
	}
	var created map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sessions/"+created["session_id"]+"/login", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", w.Code, w.Body.String())
	}
	var login map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}
	if login["verification_url"] != "https://verify.example" {
		t.Fatalf("verification_url = %q", login["verification_url"])
	}
}

func TestServerMarksLoginStartFailureAsAuthFailed(t *testing.T) {
	store := session.NewFileStore(t.TempDir())
	srv := New(ServerConfig{
		App:   config.Config{},
		Store: store,
		LoginStarter: func(ctx context.Context, s *session.Session) (string, error) {
			return "", errors.New("unable to start login")
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d body=%s", w.Code, w.Body.String())
	}

	var created map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sessions/"+created["session_id"]+"/login", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("login status = %d body=%s", w.Code, w.Body.String())
	}

	loaded, err := store.Get(created["session_id"])
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != session.StatusAuthFailed {
		t.Fatalf("status = %q, want %q", loaded.Status, session.StatusAuthFailed)
	}
	if loaded.Error != "unable to start login" {
		t.Fatalf("error = %q", loaded.Error)
	}
}

func TestServerMarksAnalyzerFailureAsAnalysisFailed(t *testing.T) {
	store := session.NewFileStore(t.TempDir())
	srv := New(ServerConfig{
		App:   config.Config{},
		Store: store,
		Analyzer: func(ctx context.Context, s *session.Session) error {
			return errors.New("analysis exploded")
		},
	})

	item, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	item.Status = session.StatusAuthenticated
	if err := store.Save(item); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+item.ID+"/analyze", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("analyze status = %d body=%s", w.Code, w.Body.String())
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		loaded, err := store.Get(item.ID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Status == session.StatusAnalysisFailed {
			if loaded.Error != "analysis exploded" {
				t.Fatalf("error = %q", loaded.Error)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	loaded, _ := store.Get(item.ID)
	t.Fatalf("session did not enter analysis_failed: %#v", loaded)
}

func TestStatusIncludesPrimaryPersonaSummaryWhenReportIsReady(t *testing.T) {
	store := session.NewFileStore(t.TempDir())
	srv := New(ServerConfig{
		App:   config.Config{},
		Store: store,
	})

	item, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	item.Status = session.StatusDone
	item.ReportHTML = filepath.Join(item.Dir, "report.html")
	item.PersonaResult = &persona.Result{
		PrimaryPersona: persona.Primary{
			Shorthand:          "PRISM",
			ChineseLabel:       "变色龙",
			ImageURL:           "/assets/photos/PRISM.png",
			ByteStyleDimension: "多元兼容",
			AnalysisDimension:  "多元文化适应力",
			OneLiner:           "多线程文化模拟器，每个频道都是真的",
		},
	}
	if err := store.Save(item); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+item.ID+"/status", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		ReportReady    bool            `json:"report_ready"`
		PrimaryPersona persona.Primary `json:"primary_persona"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.ReportReady {
		t.Fatal("report_ready = false")
	}
	if resp.PrimaryPersona.Shorthand != "PRISM" {
		t.Fatalf("primary_persona = %#v", resp.PrimaryPersona)
	}
}

func TestStatusIncludesProgressEventsAndNextAction(t *testing.T) {
	store := session.NewFileStore(t.TempDir())
	srv := New(ServerConfig{
		App:   config.Config{},
		Store: store,
	})

	item, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+item.ID+"/status", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		AppConfigRequired bool `json:"app_config_required"`
		Progress          struct {
			Stage   string `json:"stage"`
			Label   string `json:"label"`
			Percent int    `json:"percent"`
		} `json:"progress"`
		Events []struct {
			Stage string `json:"stage"`
		} `json:"events"`
		NextAction string `json:"next_action"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Progress.Stage != string(session.StatusCreated) || resp.Progress.Percent == 0 {
		t.Fatalf("progress = %#v", resp.Progress)
	}
	if len(resp.Events) == 0 || resp.Events[0].Stage != string(session.StatusCreated) {
		t.Fatalf("events = %#v", resp.Events)
	}
	if resp.NextAction != "connect_feishu" {
		t.Fatalf("next_action = %q", resp.NextAction)
	}
	if !resp.AppConfigRequired {
		t.Fatal("app_config_required = false, want true when server credentials are missing")
	}
}

func TestStatusReportsAppConfigNotRequiredWhenServerCredentialsExist(t *testing.T) {
	store := session.NewFileStore(t.TempDir())
	srv := New(ServerConfig{
		App: config.Config{
			Feishu: config.FeishuConfig{
				AppID:     "cli_fake",
				AppSecret: "fake-secret",
			},
		},
		Store: store,
	})

	item, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+item.ID+"/status", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		AppConfigRequired bool `json:"app_config_required"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.AppConfigRequired {
		t.Fatal("app_config_required = true, want false when server credentials are configured")
	}
}

func TestStatusDistinguishesAuthAndAnalysisFailures(t *testing.T) {
	store := session.NewFileStore(t.TempDir())
	srv := New(ServerConfig{
		App:   config.Config{},
		Store: store,
	})

	cases := []struct {
		name       string
		status     session.Status
		label      string
		wantAction string
	}{
		{
			name:       "auth failed",
			status:     session.Status("auth_failed"),
			label:      "授权流程失败，请重新连接飞书",
			wantAction: "complete_authorization",
		},
		{
			name:       "analysis failed",
			status:     session.Status("analysis_failed"),
			label:      "分析流程失败，可直接重试分析",
			wantAction: "start_analysis",
		},
		{
			name:       "legacy failed",
			status:     session.Status("failed"),
			label:      "流程执行失败",
			wantAction: "retry",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item, err := store.Create()
			if err != nil {
				t.Fatal(err)
			}
			item.Status = tc.status
			item.Error = "boom"
			if err := store.Save(item); err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+item.ID+"/status", nil)
			w := httptest.NewRecorder()
			srv.Router().ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status code = %d body=%s", w.Code, w.Body.String())
			}

			var resp struct {
				Status   string `json:"status"`
				Progress struct {
					Label string `json:"label"`
				} `json:"progress"`
				NextAction string `json:"next_action"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatal(err)
			}
			if resp.Status != string(tc.status) {
				t.Fatalf("status = %q, want %q", resp.Status, tc.status)
			}
			if resp.Progress.Label != tc.label {
				t.Fatalf("progress label = %q, want %q", resp.Progress.Label, tc.label)
			}
			if resp.NextAction != tc.wantAction {
				t.Fatalf("next_action = %q, want %q", resp.NextAction, tc.wantAction)
			}
		})
	}
}

func TestReportDataReturnsStructuredReport(t *testing.T) {
	store := session.NewFileStore(t.TempDir())
	srv := New(ServerConfig{
		App:   config.Config{},
		Store: store,
	})

	item, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	item.Status = session.StatusDone
	item.ReportHTML = filepath.Join(item.Dir, "report.html")
	item.PersonaResult = &persona.Result{
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
		ContrastSignals: []string{"群聊中高频输出，但在正式文档里保持高度压缩。"},
	}
	if err := store.Save(item); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+item.ID+"/report-data", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("report-data code = %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Status string         `json:"status"`
		Error  string         `json:"error"`
		Report persona.Result `json:"report"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != string(session.StatusDone) {
		t.Fatalf("status = %q", resp.Status)
	}
	if len(resp.Report.BehaviorVectors) != 6 {
		t.Fatalf("report = %#v", resp.Report)
	}
	if resp.Report.WorkProfile.ResponsibilityScope == "" {
		t.Fatalf("work_profile = %#v", resp.Report.WorkProfile)
	}
	if len(resp.Report.Analysis.Evidence) != 2 {
		t.Fatalf("evidence = %#v", resp.Report.Analysis.Evidence)
	}
	if resp.Report.InteractionInsights.RelationshipSummary == "" {
		t.Fatalf("interaction_insights = %#v", resp.Report.InteractionInsights)
	}
	if resp.Report.InteractionInsights.CoreCollaborators[0].Identifier != "" {
		t.Fatalf("core identifier leaked: %#v", resp.Report.InteractionInsights.CoreCollaborators[0])
	}
	if resp.Report.InteractionInsights.FrequentChats[0].Identifier != "" {
		t.Fatalf("chat identifier leaked: %#v", resp.Report.InteractionInsights.FrequentChats[0])
	}
	if resp.Report.Coverage.Summary == "" {
		t.Fatalf("coverage = %#v", resp.Report.Coverage)
	}
}

func TestReportDataReturnsConflictWhenReportIsNotReady(t *testing.T) {
	store := session.NewFileStore(t.TempDir())
	srv := New(ServerConfig{
		App:   config.Config{},
		Store: store,
	})

	item, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+item.ID+"/report-data", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("report-data code = %d body=%s", w.Code, w.Body.String())
	}
}

func TestWriteAnalysisPromptLogPersistsPrompt(t *testing.T) {
	dir := t.TempDir()
	prompt := "final prompt payload"
	if err := writeAnalysisPromptLog(dir, prompt); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "logs", "analysis-prompt.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != prompt {
		t.Fatalf("prompt log = %q", string(data))
	}
}

func TestWriteAnalysisDigestsLogPersistsJSON(t *testing.T) {
	dir := t.TempDir()
	input := collector.AnalysisInput{
		Stats: map[string]collector.DomainDigestStats{
			"chat_prompt": {RawChars: 120, ItemsBefore: 20, ItemsAfter: 12, FilteredItems: 8, FetchedItems: 3},
		},
	}
	if err := writeAnalysisDigestsLog(dir, input); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "logs", "analysis-domain-digests.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got collector.AnalysisInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Stats["chat_prompt"].FilteredItems != 8 {
		t.Fatalf("stats = %#v", got.Stats["chat_prompt"])
	}
}

func TestServerAutoConfiguresAppWhenCredentialsAreMissing(t *testing.T) {
	store := session.NewFileStore(t.TempDir())
	srv := New(ServerConfig{
		App: config.Config{
			LarkCLIBin: os.Args[0],
		},
		Store: store,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	var created map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sessions/"+created["session_id"]+"/login", nil)
	w = httptest.NewRecorder()
	t.Setenv("FAKE_LARK_CLI_SERVER", "1")
	t.Setenv("FAKE_CONFIG_URL", "https://config.example/page/cli")
	t.Setenv("FAKE_AUTH_URL", "https://auth.example/verify")
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", w.Code, w.Body.String())
	}
	var login map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}
	if login["verification_url"] != "https://config.example/page/cli" {
		t.Fatalf("verification_url = %q", login["verification_url"])
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		loaded, err := store.Get(created["session_id"])
		if err != nil {
			t.Fatal(err)
		}
		if loaded.VerificationURL == "https://auth.example/verify" && loaded.Status == session.StatusAuthenticated {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	loaded, _ := store.Get(created["session_id"])
	t.Fatalf("session did not advance to auth login: %#v", loaded)
}

func TestServerConfigProcessSurvivesLoginRequestContext(t *testing.T) {
	store := session.NewFileStore(t.TempDir())
	srv := New(ServerConfig{
		App: config.Config{
			LarkCLIBin: os.Args[0],
		},
		Store: store,
	})
	releaseFile := filepath.Join(t.TempDir(), "release-config")
	t.Setenv("FAKE_LARK_CLI_SERVER", "1")
	t.Setenv("FAKE_CONFIG_URL", "https://config.example/page/cli")
	t.Setenv("FAKE_AUTH_URL", "https://auth.example/verify")
	t.Setenv("FAKE_CONFIG_WAIT_FILE", releaseFile)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	var created map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	req = httptest.NewRequest(http.MethodPost, "/api/sessions/"+created["session_id"]+"/login", nil).WithContext(ctx)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", w.Code, w.Body.String())
	}
	cancel()

	time.Sleep(100 * time.Millisecond)
	loaded, err := store.Get(created["session_id"])
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status == session.StatusFailed || loaded.Status == session.StatusAuthFailed || loaded.Status == session.StatusAnalysisFailed {
		t.Fatalf("session failed after request context cancellation: %#v", loaded)
	}
	if loaded.VerificationURL != "https://config.example/page/cli" {
		t.Fatalf("verification_url = %q", loaded.VerificationURL)
	}

	if err := os.WriteFile(releaseFile, []byte("done"), 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		loaded, err = store.Get(created["session_id"])
		if err != nil {
			t.Fatal(err)
		}
		if loaded.VerificationURL == "https://auth.example/verify" && (loaded.Status == session.StatusLoginPending || loaded.Status == session.StatusAuthenticated) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	loaded, _ = store.Get(created["session_id"])
	t.Fatalf("session did not advance after config completed: %#v", loaded)
}

func TestServerSkipsOAuthWhenStoredTokenIsValid(t *testing.T) {
	testServerSkipsOAuthForTokenStatus(t, "valid")
}

func TestServerSkipsOAuthWhenStoredTokenNeedsRefresh(t *testing.T) {
	testServerSkipsOAuthForTokenStatus(t, "needs_refresh")
}

func testServerSkipsOAuthForTokenStatus(t *testing.T, tokenStatus string) {
	t.Helper()
	dataDir := t.TempDir()
	store := session.NewFileStore(dataDir)
	srv := New(ServerConfig{
		App: config.Config{
			AgentDataDir: dataDir,
			LarkCLIBin:   os.Args[0],
			Feishu: config.FeishuConfig{
				AppID:     "cli_fake",
				AppSecret: "fake-secret",
			},
		},
		Store: store,
	})
	t.Setenv("FAKE_LARK_CLI_SERVER", "1")
	t.Setenv("FAKE_AUTH_STATUS", tokenStatus)
	t.Setenv("FAKE_AUTH_LOGIN_FAIL", "1")

	req := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	var created map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sessions/"+created["session_id"]+"/login", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", w.Code, w.Body.String())
	}
	loaded, err := store.Get(created["session_id"])
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != session.StatusAuthenticated {
		t.Fatalf("status = %q, want authenticated; error=%q", loaded.Status, loaded.Error)
	}
	if loaded.VerificationURL != "" {
		t.Fatalf("verification_url = %q, want empty when token is reused", loaded.VerificationURL)
	}
}

func TestServerFallsBackToOAuthWhenStoredTokenIsExpired(t *testing.T) {
	testServerFallsBackToOAuthForTokenStatus(t, "expired")
}

func TestServerFallsBackToOAuthWhenStoredTokenIsMissing(t *testing.T) {
	testServerFallsBackToOAuthForTokenStatus(t, "")
}

func TestServerDoesNotReuseAnotherSessionsAppConfig(t *testing.T) {
	dataDir := t.TempDir()
	store := session.NewFileStore(dataDir)
	srv := New(ServerConfig{
		App: config.Config{
			AgentDataDir: dataDir,
			LarkCLIBin:   os.Args[0],
		},
		Store: store,
	})
	t.Setenv("FAKE_LARK_CLI_SERVER", "1")
	t.Setenv("FAKE_CONFIG_URL", "https://config.example/page/cli")
	t.Setenv("FAKE_AUTH_URL", "https://auth.example/verify")

	req := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	var createdA map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &createdA); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sessions/"+createdA["session_id"]+"/login", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("session A login status = %d body=%s", w.Code, w.Body.String())
	}

	var loginA map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &loginA); err != nil {
		t.Fatal(err)
	}
	if loginA["verification_url"] != "https://config.example/page/cli" {
		t.Fatalf("session A verification_url = %q", loginA["verification_url"])
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	var createdB map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &createdB); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sessions/"+createdB["session_id"]+"/login", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("session B login status = %d body=%s", w.Code, w.Body.String())
	}

	var loginB map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &loginB); err != nil {
		t.Fatal(err)
	}
	if loginB["verification_url"] != "https://config.example/page/cli" {
		t.Fatalf("session B verification_url = %q", loginB["verification_url"])
	}
}

func testServerFallsBackToOAuthForTokenStatus(t *testing.T, tokenStatus string) {
	t.Helper()
	dataDir := t.TempDir()
	store := session.NewFileStore(dataDir)
	srv := New(ServerConfig{
		App: config.Config{
			AgentDataDir: dataDir,
			LarkCLIBin:   os.Args[0],
			Feishu: config.FeishuConfig{
				AppID:     "cli_fake",
				AppSecret: "fake-secret",
			},
		},
		Store: store,
	})
	t.Setenv("FAKE_LARK_CLI_SERVER", "1")
	t.Setenv("FAKE_AUTH_STATUS", tokenStatus)
	t.Setenv("FAKE_AUTH_URL", "https://auth.example/verify")

	req := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	var created map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sessions/"+created["session_id"]+"/login", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", w.Code, w.Body.String())
	}
	var login map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}
	if login["verification_url"] != "https://auth.example/verify" {
		t.Fatalf("verification_url = %q", login["verification_url"])
	}
}

func TestServerDoesNotReuseAnotherSessionsToken(t *testing.T) {
	dataDir := t.TempDir()
	store := session.NewFileStore(dataDir)
	srv := New(ServerConfig{
		App: config.Config{
			AgentDataDir: dataDir,
			LarkCLIBin:   os.Args[0],
			Feishu: config.FeishuConfig{
				AppID:     "cli_fake",
				AppSecret: "fake-secret",
			},
		},
		Store: store,
	})
	t.Setenv("FAKE_LARK_CLI_SERVER", "1")
	t.Setenv("FAKE_AUTH_STATUS_FROM_CONFIG", "1")
	t.Setenv("FAKE_AUTH_URL", "https://auth.example/verify")

	req := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	var createdA map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &createdA); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sessions/"+createdA["session_id"]+"/login", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("session A login status = %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	var createdB map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &createdB); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sessions/"+createdB["session_id"]+"/login", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("session B login status = %d body=%s", w.Code, w.Body.String())
	}
	var login map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}
	if login["verification_url"] != "https://auth.example/verify" {
		t.Fatalf("verification_url = %q", login["verification_url"])
	}
}

func TestMain(m *testing.M) {
	if os.Getenv("FAKE_LARK_CLI_SERVER") == "1" {
		args := os.Args[1:]
		if len(args) >= 3 && args[0] == "config" && args[1] == "init" && args[2] == "--new" {
			_, _ = os.Stderr.WriteString("Open the link below to configure app:\n  " + os.Getenv("FAKE_CONFIG_URL") + "\n")
			configDir := os.Getenv("LARKSUITE_CLI_CONFIG_DIR")
			_ = os.MkdirAll(configDir, 0700)
			_ = os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"apps":[{"appId":"cli_fake","appSecret":"fake","brand":"feishu","users":[]}]}`), 0600)
			if waitFile := os.Getenv("FAKE_CONFIG_WAIT_FILE"); waitFile != "" {
				deadline := time.Now().Add(5 * time.Second)
				for time.Now().Before(deadline) {
					if _, err := os.Stat(waitFile); err == nil {
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
			os.Exit(0)
		}
		if len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
			if os.Getenv("FAKE_AUTH_STATUS_FROM_CONFIG") == "1" {
				configDir := os.Getenv("LARKSUITE_CLI_CONFIG_DIR")
				data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
				if err != nil {
					_, _ = os.Stdout.WriteString(`{"identity":"bot"}`)
					os.Exit(0)
				}
				if strings.Contains(string(data), `"tokenStatus":"valid"`) {
					_, _ = os.Stdout.WriteString(`{"identity":"user","tokenStatus":"valid"}`)
				} else {
					_, _ = os.Stdout.WriteString(`{"identity":"bot"}`)
				}
				os.Exit(0)
			}
			status := os.Getenv("FAKE_AUTH_STATUS")
			if status == "" {
				_, _ = os.Stdout.WriteString(`{"identity":"bot"}`)
				os.Exit(0)
			}
			_, _ = os.Stdout.WriteString(`{"identity":"user","tokenStatus":"` + status + `"}`)
			os.Exit(0)
		}
		if len(args) >= 5 && args[0] == "auth" && args[1] == "login" && args[4] == "--no-wait" {
			if os.Getenv("FAKE_AUTH_LOGIN_FAIL") == "1" {
				_, _ = os.Stderr.WriteString("auth login should not be called")
				os.Exit(9)
			}
			_, _ = os.Stdout.WriteString(`{"verification_url":"` + os.Getenv("FAKE_AUTH_URL") + `","device_code":"dev_fake"}`)
			os.Exit(0)
		}
		if len(args) >= 4 && args[0] == "auth" && args[1] == "login" && args[2] == "--device-code" {
			if os.Getenv("FAKE_AUTH_STATUS_FROM_CONFIG") == "1" {
				configDir := os.Getenv("LARKSUITE_CLI_CONFIG_DIR")
				path := filepath.Join(configDir, "config.json")
				data, err := os.ReadFile(path)
				if err == nil {
					updated := strings.Replace(string(data), `"brand":"feishu"`, `"brand":"feishu","users":[{"identity":"user","tokenStatus":"valid","accessToken":"access","refreshToken":"refresh"}]`, 1)
					_ = os.WriteFile(path, []byte(updated), 0600)
				}
			}
			os.Exit(0)
		}
		os.Exit(2)
	}
	os.Exit(m.Run())
}
