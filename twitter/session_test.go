package twitter

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestSessionStore_AddAndGet(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store := NewSessionStore(tmpFile.Name())

	session := Session{
		Username:  "testuser",
		AuthToken: "auth_token_123",
		CT0:       "ct0_456",
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		Active:    true,
	}

	err = store.Add(session)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	sessions := store.GetAll()
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Username != "testuser" {
		t.Errorf("Username = %q, want %q", sessions[0].Username, "testuser")
	}
	if sessions[0].AuthToken != "auth_token_123" {
		t.Errorf("AuthToken = %q, want %q", sessions[0].AuthToken, "auth_token_123")
	}
}

func TestSessionStore_Persistence(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store1 := NewSessionStore(tmpFile.Name())
	store1.Add(Session{Username: "user1", AuthToken: "tok1", CT0: "ct01", Active: true, CreatedAt: time.Now(), LastUsed: time.Now()})
	store1.Add(Session{Username: "user2", AuthToken: "tok2", CT0: "ct02", Active: true, CreatedAt: time.Now(), LastUsed: time.Now()})

	// Create a new store from the same file to verify persistence
	store2 := NewSessionStore(tmpFile.Name())
	sessions := store2.GetAll()

	if len(sessions) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].Username != "user1" || sessions[1].Username != "user2" {
		t.Error("Sessions not persisted correctly")
	}
}

func TestSessionStore_GetActive(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store := NewSessionStore(tmpFile.Name())
	store.Add(Session{Username: "active", AuthToken: "tok1", CT0: "ct01", Active: true, CreatedAt: time.Now(), LastUsed: time.Now()})
	store.Add(Session{Username: "inactive", AuthToken: "tok2", CT0: "ct02", Active: false, CreatedAt: time.Now(), LastUsed: time.Now()})
	store.Add(Session{Username: "errored", AuthToken: "tok3", CT0: "ct03", Active: true, Error: "expired", CreatedAt: time.Now(), LastUsed: time.Now()})

	active := store.GetActive()
	if len(active) != 1 {
		t.Fatalf("Expected 1 active session, got %d", len(active))
	}
	if active[0].Username != "active" {
		t.Errorf("Expected active session username = 'active', got %q", active[0].Username)
	}
}

func TestSessionStore_Remove(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store := NewSessionStore(tmpFile.Name())
	store.Add(Session{Username: "user1", AuthToken: "tok1", CT0: "ct01", Active: true, CreatedAt: time.Now(), LastUsed: time.Now()})
	store.Add(Session{Username: "user2", AuthToken: "tok2", CT0: "ct02", Active: true, CreatedAt: time.Now(), LastUsed: time.Now()})

	store.Remove("user1")

	sessions := store.GetAll()
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session after remove, got %d", len(sessions))
	}
	if sessions[0].Username != "user2" {
		t.Errorf("Wrong session remaining: %q", sessions[0].Username)
	}
}

func TestSessionStore_Toggle(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store := NewSessionStore(tmpFile.Name())
	store.Add(Session{Username: "user1", AuthToken: "tok1", CT0: "ct01", Active: true, CreatedAt: time.Now(), LastUsed: time.Now()})

	store.Toggle("user1")
	sessions := store.GetAll()
	if sessions[0].Active {
		t.Error("Expected session to be inactive after toggle")
	}

	store.Toggle("user1")
	sessions = store.GetAll()
	if !sessions[0].Active {
		t.Error("Expected session to be active after second toggle")
	}
}

func TestSessionStore_ReplaceExisting(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store := NewSessionStore(tmpFile.Name())
	store.Add(Session{Username: "user1", AuthToken: "old_token", CT0: "old_ct0", Active: true, CreatedAt: time.Now(), LastUsed: time.Now()})
	store.Add(Session{Username: "user1", AuthToken: "new_token", CT0: "new_ct0", Active: true, CreatedAt: time.Now(), LastUsed: time.Now()})

	sessions := store.GetAll()
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session (replaced), got %d", len(sessions))
	}
	if sessions[0].AuthToken != "new_token" {
		t.Errorf("AuthToken = %q, want %q", sessions[0].AuthToken, "new_token")
	}
}

func TestSessionJSONL_Format(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store := NewSessionStore(tmpFile.Name())
	now := time.Now()
	store.Add(Session{Username: "user1", AuthToken: "tok1", CT0: "ct01", Active: true, CreatedAt: now, LastUsed: now})

	// Read the file and verify it's valid JSONL
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	var session Session
	err = json.Unmarshal(data[:len(data)-1], &session) // Remove trailing newline
	if err != nil {
		t.Fatalf("Failed to parse JSONL line: %v", err)
	}
	if session.Username != "user1" {
		t.Errorf("Username = %q, want %q", session.Username, "user1")
	}
}
