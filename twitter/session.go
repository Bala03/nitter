package twitter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Session represents a Twitter authenticated session stored in session.jsonl
type Session struct {
	Username    string    `json:"username"`
	AuthToken   string    `json:"auth_token"`
	CT0         string    `json:"ct0"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used"`
	Active      bool      `json:"active"`
	DisplayName string    `json:"display_name,omitempty"`
	UserID      string    `json:"user_id,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// SessionStore manages reading/writing sessions from session.jsonl
type SessionStore struct {
	filePath string
	mu       sync.RWMutex
	sessions []Session
}

func NewSessionStore(filePath string) *SessionStore {
	store := &SessionStore{filePath: filePath}
	store.load()
	return store
}

func (s *SessionStore) load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.filePath)
	if err != nil {
		s.sessions = []Session{}
		return
	}
	defer file.Close()

	s.sessions = nil
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var session Session
		if err := json.Unmarshal([]byte(line), &session); err != nil {
			continue
		}
		s.sessions = append(s.sessions, session)
	}
}

func (s *SessionStore) save() error {
	s.mu.RLock()
	sessions := make([]Session, len(s.sessions))
	copy(sessions, s.sessions)
	s.mu.RUnlock()

	file, err := os.Create(s.filePath)
	if err != nil {
		return fmt.Errorf("create session file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, session := range sessions {
		data, err := json.Marshal(session)
		if err != nil {
			continue
		}
		writer.Write(data)
		writer.WriteByte('\n')
	}
	return writer.Flush()
}

// Add adds a new session to the store
func (s *SessionStore) Add(session Session) error {
	s.mu.Lock()
	// Replace existing session for same username
	found := false
	for i, existing := range s.sessions {
		if existing.Username == session.Username {
			s.sessions[i] = session
			found = true
			break
		}
	}
	if !found {
		s.sessions = append(s.sessions, session)
	}
	s.mu.Unlock()

	return s.save()
}

// GetAll returns all sessions
func (s *SessionStore) GetAll() []Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Session, len(s.sessions))
	copy(result, s.sessions)
	return result
}

// GetActive returns all active sessions
func (s *SessionStore) GetActive() []Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var active []Session
	for _, session := range s.sessions {
		if session.Active && session.Error == "" {
			active = append(active, session)
		}
	}
	return active
}

// Remove removes a session by username
func (s *SessionStore) Remove(username string) error {
	s.mu.Lock()
	for i, session := range s.sessions {
		if session.Username == username {
			s.sessions = append(s.sessions[:i], s.sessions[i+1:]...)
			break
		}
	}
	s.mu.Unlock()
	return s.save()
}

// Toggle enables or disables a session
func (s *SessionStore) Toggle(username string) error {
	s.mu.Lock()
	for i, session := range s.sessions {
		if session.Username == username {
			s.sessions[i].Active = !s.sessions[i].Active
			break
		}
	}
	s.mu.Unlock()
	return s.save()
}

// UpdateLastUsed marks a session as recently used
func (s *SessionStore) UpdateLastUsed(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, session := range s.sessions {
		if session.Username == username {
			s.sessions[i].LastUsed = time.Now()
			break
		}
	}
	// fire-and-forget save
	go s.save()
}

// MarkError marks a session as having an error
func (s *SessionStore) MarkError(username string, errMsg string) {
	s.mu.Lock()
	for i, session := range s.sessions {
		if session.Username == username {
			s.sessions[i].Error = errMsg
			s.sessions[i].Active = false
			break
		}
	}
	s.mu.Unlock()
	s.save()
}
