package state

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	// LastRalphEpochStart tracks the mtime of RALPH-LOOP.lock when we last
	// initialised a ralph epoch. When the lock is recreated (new loop started),
	// its mtime will be newer and we reset CompactionCount for the fresh loop.
	LastRalphEpochStart time.Time `json:"last_ralph_epoch_start,omitempty"`

	baseDir   string
	sessionID string
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
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.stateFile(), data, 0644)
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
