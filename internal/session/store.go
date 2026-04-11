package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"feishu-personality-agent/internal/persona"
)

type Status string

const (
	StatusCreated       Status = "created"
	StatusConfigPending Status = "config_pending"
	StatusLoginPending  Status = "login_pending"
	StatusAuthenticated Status = "authenticated"
	StatusCollecting    Status = "collecting"
	StatusAnalyzing     Status = "analyzing"
	StatusDone          Status = "done"
	StatusFailed        Status = "failed"
)

type Session struct {
	ID              string          `json:"id"`
	Status          Status          `json:"status"`
	Dir             string          `json:"dir"`
	VerificationURL string          `json:"verification_url,omitempty"`
	DeviceCode      string          `json:"device_code,omitempty"`
	Error           string          `json:"error,omitempty"`
	ReportMarkdown  string          `json:"report_markdown,omitempty"`
	ReportHTML      string          `json:"report_html,omitempty"`
	PersonaResult   *persona.Result `json:"persona_result,omitempty"`
	Events          []Event         `json:"events,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type Event struct {
	Stage     string    `json:"stage"`
	Label     string    `json:"label"`
	Percent   int       `json:"percent"`
	Timestamp time.Time `json:"timestamp"`
}

type FileStore struct {
	baseDir string
}

func NewFileStore(baseDir string) *FileStore {
	if abs, err := filepath.Abs(filepath.Clean(baseDir)); err == nil {
		baseDir = abs
	} else {
		baseDir = filepath.Clean(baseDir)
	}
	return &FileStore{baseDir: baseDir}
}

func (s *FileStore) Create() (*Session, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	sessionDir := filepath.Join(s.baseDir, "sessions", id)
	session := &Session{
		ID:        id,
		Status:    StatusCreated,
		Dir:       sessionDir,
		CreatedAt: now,
		UpdatedAt: now,
	}
	session.RecordEvent(StatusCreated, "Session created", 5)
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, err
	}
	if err := s.Save(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *FileStore) Get(id string) (*Session, error) {
	data, err := os.ReadFile(filepath.Join(s.baseDir, "sessions", id, "session.json"))
	if err != nil {
		return nil, err
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *FileStore) Save(session *Session) error {
	session.UpdatedAt = time.Now().UTC()
	if err := os.MkdirAll(session.Dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(session.Dir, "session.json"), append(data, '\n'), 0600)
}

func (s *Session) RecordEvent(stage Status, label string, percent int) {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	event := Event{
		Stage:     string(stage),
		Label:     label,
		Percent:   percent,
		Timestamp: time.Now().UTC(),
	}
	if n := len(s.Events); n > 0 {
		last := s.Events[n-1]
		if last.Stage == event.Stage && last.Label == event.Label && last.Percent == event.Percent {
			return
		}
	}
	s.Events = append(s.Events, event)
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
