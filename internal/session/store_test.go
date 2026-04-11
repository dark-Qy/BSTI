package session

import (
	"path/filepath"
	"testing"
)

func TestStoreCreatesAndPersistsSession(t *testing.T) {
	store := NewFileStore(t.TempDir())
	s, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != StatusCreated {
		t.Fatalf("status = %q", s.Status)
	}

	s.Status = StatusAuthenticated
	if err := store.Save(s); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Get(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != StatusAuthenticated {
		t.Fatalf("loaded status = %q", loaded.Status)
	}
	if loaded.Dir == "" {
		t.Fatal("session dir is empty")
	}
}

func TestStoreNormalizesRelativeBaseDirToAbsoluteSessionDir(t *testing.T) {
	t.Chdir(t.TempDir())
	store := NewFileStore("data")
	s, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(s.Dir) {
		t.Fatalf("session dir = %q, want absolute path", s.Dir)
	}
	if got := filepath.Clean(s.Dir); got != s.Dir {
		t.Fatalf("session dir = %q, want clean path %q", s.Dir, got)
	}
}

func TestStoreCreateSeedsCreatedEvent(t *testing.T) {
	store := NewFileStore(t.TempDir())
	s, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Events) != 1 {
		t.Fatalf("events = %#v", s.Events)
	}
	if s.Events[0].Stage != string(StatusCreated) {
		t.Fatalf("event stage = %q", s.Events[0].Stage)
	}
}

func TestRecordEventSkipsConsecutiveDuplicates(t *testing.T) {
	s := &Session{}
	s.RecordEvent(StatusCreated, "Session created", 5)
	s.RecordEvent(StatusCreated, "Session created", 5)
	s.RecordEvent(StatusAuthenticated, "Feishu authorization completed", 45)
	if len(s.Events) != 2 {
		t.Fatalf("events = %#v", s.Events)
	}
}
