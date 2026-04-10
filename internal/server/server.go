package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"

	"feishu-personality-agent/internal/collector"
	"feishu-personality-agent/internal/config"
	"feishu-personality-agent/internal/llm"
	"feishu-personality-agent/internal/persona"
	"feishu-personality-agent/internal/report"
	"feishu-personality-agent/internal/sandbox"
	"feishu-personality-agent/internal/session"
)

const readOnlyScopes = "search:message contact:user.basic_profile:readonly search:docs:read docx:document:readonly calendar:calendar.event:read task:task:read mail:user_mailbox.message:readonly mail:user_mailbox.message.address:read mail:user_mailbox.message.subject:read mail:user_mailbox.message.body:read vc:meeting.search:read vc:meeting.meetingevent:read vc:note:read"

var urlPattern = regexp.MustCompile(`https?://\S+`)

type LoginStarter func(context.Context, *session.Session) (string, error)
type Analyzer func(context.Context, *session.Session) error

type ServerConfig struct {
	App          config.Config
	Store        *session.FileStore
	LoginStarter LoginStarter
	Analyzer     Analyzer
}

type Server struct {
	cfg          config.Config
	store        *session.FileStore
	loginStarter LoginStarter
	analyzer     Analyzer
	router       *gin.Engine
}

func New(cfg ServerConfig) *Server {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{
		cfg:          cfg.App,
		store:        cfg.Store,
		loginStarter: cfg.LoginStarter,
		analyzer:     cfg.Analyzer,
	}
	if s.loginStarter == nil {
		s.loginStarter = s.startLogin
	}
	if s.analyzer == nil {
		s.analyzer = s.runAnalysis
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.StaticFS("/assets/photos", gin.Dir("photos", false))
	r.GET("/", s.index)
	r.POST("/api/sessions", s.createSession)
	r.POST("/api/sessions/:id/login", s.login)
	r.GET("/api/sessions/:id/status", s.status)
	r.POST("/api/sessions/:id/analyze", s.analyze)
	r.GET("/api/sessions/:id/report", s.report)
	s.router = r
	return s
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) index(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(indexHTML))
}

func (s *Server) createSession(c *gin.Context) {
	item, err := s.store.Create()
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"session_id": item.ID, "status": item.Status})
}

func (s *Server) login(c *gin.Context) {
	item, err := s.store.Get(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err)
		return
	}
	url, err := s.loginStarter(c.Request.Context(), item)
	if err != nil {
		item.Status = session.StatusFailed
		item.Error = err.Error()
		_ = s.store.Save(item)
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"verification_url": url, "status": item.Status})
}

func (s *Server) status(c *gin.Context) {
	item, err := s.store.Get(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"session_id":       item.ID,
		"status":           item.Status,
		"verification_url": item.VerificationURL,
		"error":            item.Error,
		"report_ready":     item.ReportHTML != "",
		"primary_persona":  primaryPersonaSummary(item),
	})
}

func (s *Server) analyze(c *gin.Context) {
	item, err := s.store.Get(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err)
		return
	}
	if item.Status != session.StatusAuthenticated && item.Status != session.StatusDone {
		c.JSON(http.StatusConflict, gin.H{"error": "session is not authenticated", "status": item.Status})
		return
	}
	item.Status = session.StatusCollecting
	item.Error = ""
	item.PersonaResult = nil
	if err := s.store.Save(item); err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	go func() {
		if err := s.analyzer(context.Background(), item); err != nil {
			item.Status = session.StatusFailed
			item.Error = err.Error()
			_ = s.store.Save(item)
		}
	}()
	c.JSON(http.StatusOK, gin.H{"status": item.Status})
}

func (s *Server) report(c *gin.Context) {
	item, err := s.store.Get(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err)
		return
	}
	if item.ReportHTML == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "report is not ready"})
		return
	}
	c.File(item.ReportHTML)
}

func (s *Server) startLogin(ctx context.Context, item *session.Session) (string, error) {
	seeded, err := s.seedSessionCLIConfig(item)
	if err != nil {
		return "", err
	}
	if seeded {
		reused, err := s.tryReuseLogin(ctx, item)
		if err != nil {
			return "", err
		}
		if reused {
			return "", nil
		}
	}
	if s.cfg.Feishu.AppID == "" || s.cfg.Feishu.AppSecret == "" {
		return s.startConfigThenLogin(ctx, item)
	}
	if !seeded {
		if err := writeCLIConfig(item.Dir, s.cfg.Feishu); err != nil {
			return "", err
		}
	}
	reused, err := s.tryReuseLogin(ctx, item)
	if err != nil {
		return "", err
	}
	if reused {
		return "", nil
	}
	return s.startOAuthLogin(ctx, item)
}

func (s *Server) seedSessionCLIConfig(item *session.Session) (bool, error) {
	src := filepath.Join(s.sharedCLIConfigDir(item), "config.json")
	data, err := os.ReadFile(src)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	dstDir := sandbox.NewExecutor(s.cfg.LarkCLIBin, item.Dir, 5*time.Minute).ConfigDir()
	if err := os.MkdirAll(dstDir, 0700); err != nil {
		return false, err
	}
	if err := os.WriteFile(filepath.Join(dstDir, "config.json"), data, 0600); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Server) persistSessionCLIConfig(item *session.Session) error {
	src := filepath.Join(sandbox.NewExecutor(s.cfg.LarkCLIBin, item.Dir, 5*time.Minute).ConfigDir(), "config.json")
	data, err := os.ReadFile(src)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	dstDir := s.sharedCLIConfigDir(item)
	if err := os.MkdirAll(dstDir, 0700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dstDir, "config.json"), data, 0600)
}

func (s *Server) sharedCLIConfigDir(item *session.Session) string {
	dataDir := s.cfg.AgentDataDir
	if dataDir == "" {
		dataDir = filepath.Dir(filepath.Dir(item.Dir))
	}
	if abs, err := filepath.Abs(filepath.Clean(dataDir)); err == nil {
		dataDir = abs
	} else {
		dataDir = filepath.Clean(dataDir)
	}
	return filepath.Join(dataDir, "lark-cli")
}

func (s *Server) tryReuseLogin(ctx context.Context, item *session.Session) (bool, error) {
	exe := sandbox.NewExecutor(s.cfg.LarkCLIBin, item.Dir, 30*time.Second)
	out, err := exe.Run(ctx, []string{"auth", "status"})
	if err != nil {
		return false, nil
	}
	var status struct {
		Identity    string `json:"identity"`
		TokenStatus string `json:"tokenStatus"`
	}
	if err := json.Unmarshal([]byte(out.Stdout), &status); err != nil {
		return false, nil
	}
	if status.Identity != "user" || (status.TokenStatus != "valid" && status.TokenStatus != "needs_refresh") {
		return false, nil
	}
	item.Status = session.StatusAuthenticated
	item.VerificationURL = ""
	item.DeviceCode = ""
	item.Error = ""
	return true, s.store.Save(item)
}

func (s *Server) startConfigThenLogin(ctx context.Context, item *session.Session) (string, error) {
	exe := sandbox.NewExecutor(s.cfg.LarkCLIBin, item.Dir, 10*time.Minute)
	if err := exe.Validate([]string{"config", "init", "--new"}); err != nil {
		return "", err
	}
	if err := os.MkdirAll(exe.ConfigDir(), 0700); err != nil {
		return "", err
	}
	processCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	cmd := exec.CommandContext(processCtx, s.cfg.LarkCLIBin, "config", "init", "--new")
	cmd.Dir = item.Dir
	cmd.Env = append(os.Environ(), "LARKSUITE_CLI_CONFIG_DIR="+exe.ConfigDir())
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return "", err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return "", err
	}

	urlCh := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			if found := urlPattern.FindString(scanner.Text()); found != "" {
				select {
				case urlCh <- found:
				default:
				}
			}
		}
	}()

	select {
	case configURL := <-urlCh:
		item.Status = session.StatusConfigPending
		item.VerificationURL = configURL
		if err := s.store.Save(item); err != nil {
			cancel()
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return "", err
		}
		go s.waitConfigThenStartOAuth(cmd, cancel, item.ID)
		return configURL, nil
	case <-time.After(30 * time.Second):
		cancel()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return "", errors.New("timed out waiting for app configuration URL")
	case <-ctx.Done():
		cancel()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return "", ctx.Err()
	}
}

func (s *Server) waitConfigThenStartOAuth(cmd *exec.Cmd, cancel context.CancelFunc, sessionID string) {
	defer cancel()
	if err := cmd.Wait(); err != nil {
		current, loadErr := s.store.Get(sessionID)
		if loadErr == nil {
			current.Status = session.StatusFailed
			current.Error = err.Error()
			_ = s.store.Save(current)
		}
		return
	}
	current, err := s.store.Get(sessionID)
	if err != nil {
		return
	}
	if err := s.persistSessionCLIConfig(current); err != nil {
		current.Status = session.StatusFailed
		current.Error = err.Error()
		_ = s.store.Save(current)
		return
	}
	if _, err := s.startOAuthLogin(context.Background(), current); err != nil {
		current.Status = session.StatusFailed
		current.Error = err.Error()
		_ = s.store.Save(current)
	}
}

func (s *Server) startOAuthLogin(ctx context.Context, item *session.Session) (string, error) {
	exe := sandbox.NewExecutor(s.cfg.LarkCLIBin, item.Dir, 5*time.Minute)
	out, err := exe.Run(ctx, []string{"auth", "login", "--scope", readOnlyScopes, "--no-wait"})
	if err != nil {
		return "", err
	}
	var resp struct {
		VerificationURL string `json:"verification_url"`
		DeviceCode      string `json:"device_code"`
	}
	if err := json.Unmarshal([]byte(out.Stdout), &resp); err != nil {
		return "", err
	}
	item.Status = session.StatusLoginPending
	item.VerificationURL = resp.VerificationURL
	item.DeviceCode = resp.DeviceCode
	if err := s.store.Save(item); err != nil {
		return "", err
	}
	go func() {
		pollOut, pollErr := exe.Run(context.Background(), []string{"auth", "login", "--device-code", resp.DeviceCode})
		current, err := s.store.Get(item.ID)
		if err != nil {
			return
		}
		if pollErr != nil {
			current.Status = session.StatusFailed
			current.Error = pollErr.Error()
			_ = pollOut
		} else {
			current.Status = session.StatusAuthenticated
			current.Error = ""
			if err := s.persistSessionCLIConfig(current); err != nil {
				current.Status = session.StatusFailed
				current.Error = err.Error()
			}
		}
		_ = s.store.Save(current)
	}()
	return resp.VerificationURL, nil
}

func (s *Server) runAnalysis(ctx context.Context, item *session.Session) error {
	if s.cfg.AIDP.AK == "" {
		return errors.New("missing AIDP_AK in .env")
	}
	exe := sandbox.NewExecutor(s.cfg.LarkCLIBin, item.Dir, 5*time.Minute)
	bundle, err := collector.New(exe).Collect(ctx, item.Dir, time.Now())
	if err != nil {
		return err
	}
	item.Status = session.StatusAnalyzing
	if err := s.store.Save(item); err != nil {
		return err
	}
	catalog := persona.All()
	prompt := collector.BuildAnalysisPrompt(bundle, catalog)
	client := llm.NewAIDPClient(llm.Config{
		URL:       s.cfg.AIDP.ModelHubURL,
		AK:        s.cfg.AIDP.AK,
		Model:     s.cfg.AIDP.Model,
		MaxTokens: s.cfg.AIDP.MaxTokens,
		Stream:    s.cfg.AIDP.Stream,
	}, http.DefaultClient)
	output, err := client.GenerateWithValidation(ctx, prompt, func(content string) error {
		_, parseErr := persona.ParseLLMResult(content)
		return parseErr
	})
	if err != nil {
		return err
	}
	result, err := persona.ParseLLMResult(output)
	if err != nil {
		return err
	}
	paths, err := report.WriteLocal(item.Dir, result)
	if err != nil {
		return err
	}
	item.PersonaResult = &result
	item.ReportMarkdown = paths.Markdown
	item.ReportHTML = paths.HTML
	item.Status = session.StatusDone
	return s.store.Save(item)
}

func writeCLIConfig(sessionDir string, feishu config.FeishuConfig) error {
	cliDir := filepath.Join(sessionDir, "lark-cli")
	if err := os.MkdirAll(cliDir, 0700); err != nil {
		return err
	}
	secretPath := filepath.Join(cliDir, "app_secret")
	if err := os.WriteFile(secretPath, []byte(feishu.AppSecret), 0600); err != nil {
		return err
	}
	cfg := map[string]any{
		"apps": []map[string]any{
			{
				"appId": feishu.AppID,
				"appSecret": map[string]string{
					"source": "file",
					"id":     secretPath,
				},
				"brand": "feishu",
				"lang":  "zh",
				"users": []any{},
			},
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cliDir, "config.json"), append(data, '\n'), 0600)
}

func writeError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{"error": err.Error()})
}

func primaryPersonaSummary(item *session.Session) any {
	if item == nil || item.PersonaResult == nil {
		return nil
	}
	return item.PersonaResult.PrimaryPersona
}

var indexHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Feishu Personality Agent</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; max-width: 880px; margin: 40px auto; padding: 0 20px; line-height: 1.6; }
    button { border-radius: 8px; padding: 8px 14px; border: 1px solid #222; background: #fff; cursor: pointer; }
    code, pre { background: #f5f5f5; padding: 2px 4px; }
  </style>
</head>
<body>
  <h1>Feishu Personality Agent</h1>
  <p>Authorize Feishu access, collect recent read-only data, and generate a local MBTI-like behavior report.</p>
  <button onclick="createSession()">Start</button>
  <button onclick="startLogin()">Login</button>
  <button onclick="analyze()">Analyze</button>
  <p id="status">No session yet.</p>
  <p id="link"></p>
  <div id="persona" style="display:none"></div>
  <p><a id="report" href="#" target="_blank" style="display:none">Open report</a></p>
  <script>
    let sessionID = "";
    async function createSession() {
      const res = await fetch("/api/sessions", {method: "POST"});
      const data = await res.json();
      sessionID = data.session_id;
      document.getElementById("status").textContent = "Session: " + sessionID + " (" + data.status + ")";
    }
    async function startLogin() {
      if (!sessionID) await createSession();
      const res = await fetch("/api/sessions/" + sessionID + "/login", {method: "POST"});
      const data = await res.json();
      if (data.verification_url) {
        document.getElementById("link").innerHTML = '<a href="' + data.verification_url + '" target="_blank">Open Feishu login link</a>';
      }
      poll();
    }
    async function analyze() {
      const res = await fetch("/api/sessions/" + sessionID + "/analyze", {method: "POST"});
      const data = await res.json();
      document.getElementById("status").textContent = JSON.stringify(data);
      poll();
    }
    async function poll() {
      if (!sessionID) return;
      const res = await fetch("/api/sessions/" + sessionID + "/status");
      const data = await res.json();
      document.getElementById("status").textContent = JSON.stringify(data);
      if (data.status === "authenticated") {
        document.getElementById("link").textContent = "Feishu authorization completed. Click Analyze to generate the report.";
      } else if (data.verification_url) {
        document.getElementById("link").innerHTML = '<a href="' + data.verification_url + '" target="_blank">Open current Feishu link</a>';
      }
      if (data.report_ready) {
        const a = document.getElementById("report");
        a.href = "/api/sessions/" + sessionID + "/report";
        a.style.display = "inline";
      }
      if (data.primary_persona) {
        const card = document.getElementById("persona");
        card.style.display = "block";
        card.innerHTML = '<div style="display:grid;grid-template-columns:120px 1fr;gap:16px;align-items:center;border:1px solid #ddd;border-radius:8px;padding:16px;margin:16px 0"><img src="' + data.primary_persona.image_url + '" alt="' + data.primary_persona.shorthand + '" style="width:120px;height:auto;border-radius:8px"><div><div style="font-size:12px;color:#666">BSPI Top1 Persona</div><div style="font-weight:700;font-size:20px">' + data.primary_persona.shorthand + ' / ' + data.primary_persona.chinese_label + '</div><div style="margin-top:6px">' + data.primary_persona.one_liner + '</div></div></div>';
      }
      if (!["done", "failed"].includes(data.status)) setTimeout(poll, 2000);
    }
  </script>
</body>
</html>`
