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
	"feishu-personality-agent/internal/session"
)

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
		if loaded.VerificationURL == "https://auth.example/verify" && (loaded.Status == session.StatusLoginPending || loaded.Status == session.StatusAuthenticated) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	loaded, _ := store.Get(created["session_id"])
	t.Fatalf("session did not advance to auth login: %#v", loaded)
}

func TestMain(m *testing.M) {
	if os.Getenv("FAKE_LARK_CLI_SERVER") == "1" {
		args := os.Args[1:]
		if len(args) >= 3 && args[0] == "config" && args[1] == "init" && args[2] == "--new" {
			_, _ = os.Stderr.WriteString("Open the link below to configure app:\n  " + os.Getenv("FAKE_CONFIG_URL") + "\n")
			configDir := os.Getenv("LARKSUITE_CLI_CONFIG_DIR")
			_ = os.MkdirAll(configDir, 0700)
			_ = os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"apps":[{"appId":"cli_fake","appSecret":"fake","brand":"feishu","users":[]}]}`), 0600)
			os.Exit(0)
		}
		if len(args) >= 5 && args[0] == "auth" && args[1] == "login" && args[4] == "--no-wait" {
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
