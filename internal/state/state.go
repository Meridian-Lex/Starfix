package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SessionState holds per-session compaction and escalation state.
type SessionState struct {
	CompactionCount   int       `json:"compaction_count"`
	EscalationPending bool      `json:"escalation_pending"`
	TriageDefault     string    `json:"triage_default"`
	EscalationSentAt  time.Time `json:"escalation_sent_at,omitempty"`
	ReplyReceived     bool      `json:"reply_received"`
	ReplyText         string    `json:"reply_text"`
	TimeoutFired      bool      `json:"timeout_fired"`
	TimeoutAction     string    `json:"timeout_action"`

	baseDir   string
	sessionID string
	mu        sync.Mutex `json:"-"`
}

// Load reads session state from disk, returning defaults if no file exists.
func Load(baseDir, sessionID string) (*SessionState, error) {
	s := &SessionState{baseDir: baseDir, sessionID: sessionID}
	if err := os.MkdirAll(s.Dir(), 0755); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(s.stateFile())
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	s.baseDir = baseDir
	s.sessionID = sessionID
	return s, nil
}

// Save writes session state to disk.
func (s *SessionState) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.stateFile(), data, 0644)
}

// IncrementCompactionCount atomically increments the compaction count and saves state.
func (s *SessionState) IncrementCompactionCount() error {
	s.mu.Lock()
	s.CompactionCount++
	s.mu.Unlock()
	return s.Save()
}

// ResetCompactionCount sets the compaction count to zero and saves state.
func (s *SessionState) ResetCompactionCount() error {
	s.mu.Lock()
	s.CompactionCount = 0
	s.mu.Unlock()
	return s.Save()
}

// StateFile returns the path to the state.json file.
func (s *SessionState) StateFile() string {
	return s.stateFile()
}

// Dir returns the session-specific directory.
func (s *SessionState) Dir() string {
	return filepath.Join(s.baseDir, "sessions", s.sessionID)
}

func (s *SessionState) stateFile() string {
	return filepath.Join(s.Dir(), "state.json")
}

func (s *SessionState) markerFile() string {
	return filepath.Join(s.Dir(), "compact-pending")
}

// MarkerFile returns the path to the compact-pending marker file.
func (s *SessionState) MarkerFile() string {
	return s.markerFile()
}

// WriteMarker creates the compact-pending marker file.
func (s *SessionState) WriteMarker() error {
	return os.WriteFile(s.markerFile(), []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
}

// MarkerExists reports whether the compact-pending marker is present.
func (s *SessionState) MarkerExists() bool {
	_, err := os.Stat(s.markerFile())
	return err == nil
}

// DeleteMarker removes the compact-pending marker file.
func (s *SessionState) DeleteMarker() {
	os.Remove(s.markerFile())
}

// DefaultBaseDir returns ~/.config/starfix
func DefaultBaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "starfix")
}
