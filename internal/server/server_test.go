package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		Progress struct {
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
			Summary:            "跨文化语境切换自然，适合做协作桥梁。",
			Evidence:           []string{"频繁在不同协作对象之间切换表达方式", "跨团队沟通密度高"},
			CommunicationStyle: "先理解对方语境，再翻译回共同问题。",
			WorkPreferences:    "偏好多方协作与需要桥接认知差异的任务。",
			BlindSpots:         "可能长期适配别人而忽略自己的固定表达方式。",
			Confidence:         0.86,
			Disclaimer:         "仅基于授权数据的行为风格观察。",
		},
		HighlightTags: []string{"跨团队桥接", "语境切换", "协作雷达"},
		BehaviorVectors: []persona.BehaviorVector{
			{Label: "协作方式", LeftPole: "独立成局", RightPole: "高频协同", Score: 82, Summary: "多人协同场景更强。"},
			{Label: "表达风格", LeftPole: "克制压缩", RightPole: "高频输出", Score: 71, Summary: "输出密度较高。"},
			{Label: "决策路径", LeftPole: "证据校准", RightPole: "直觉快判", Score: 44, Summary: "先校准事实再下判断。"},
			{Label: "推进节奏", LeftPole: "稳态推进", RightPole: "高压突进", Score: 58, Summary: "稳中偏快。"},
		},
		Coverage: persona.Coverage{
			SuccessfulDomains: []string{"chat", "docs", "calendar"},
			FailedDomains:     []string{"mail"},
			Summary:           "已覆盖 3 个数据域，1 个数据域因权限受限未纳入。",
		},
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
	if len(resp.Report.BehaviorVectors) != 4 {
		t.Fatalf("report = %#v", resp.Report)
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
	if loaded.Status == session.StatusFailed {
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

func TestServerReusesStableProfileWhenCredentialsAreMissing(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "lark-cli"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "lark-cli", "config.json"), []byte(`{"apps":[{"appId":"cli_fake","appSecret":"fake","brand":"feishu","users":[{"userOpenId":"ou_fake","userName":"Fake User"}]}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	store := session.NewFileStore(dataDir)
	srv := New(ServerConfig{
		App: config.Config{
			AgentDataDir: dataDir,
			LarkCLIBin:   os.Args[0],
		},
		Store: store,
	})
	t.Setenv("FAKE_LARK_CLI_SERVER", "1")
	t.Setenv("FAKE_AUTH_STATUS", "valid")
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
		t.Fatalf("status = %q, want authenticated", loaded.Status)
	}
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
			os.Exit(0)
		}
		os.Exit(2)
	}
	os.Exit(m.Run())
}
