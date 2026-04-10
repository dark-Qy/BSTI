package session

import "testing"

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
