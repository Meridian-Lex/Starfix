package state_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/meridian-lex/starfix/internal/state"
)

func TestState_LoadEmpty(t *testing.T) {
	dir := t.TempDir()
	s, err := state.Load(dir, "session-abc")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s.CompactionCount != 0 {
		t.Errorf("CompactionCount: got %d, want 0", s.CompactionCount)
	}
	if s.EscalationPending {
		t.Error("EscalationPending: got true, want false")
	}
}

func TestState_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	s, _ := state.Load(dir, "session-abc")
	s.CompactionCount = 3
	s.EscalationPending = true
	s.TriageDefault = "park"
	s.EscalationSentAt = time.Now().UTC()

	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	s2, err := state.Load(dir, "session-abc")
	if err != nil {
		t.Fatalf("Load after save failed: %v", err)
	}
	if s2.CompactionCount != 3 {
		t.Errorf("CompactionCount: got %d, want 3", s2.CompactionCount)
	}
	if !s2.EscalationPending {
		t.Error("EscalationPending: got false, want true")
	}
	if s2.TriageDefault != "park" {
		t.Errorf("TriageDefault: got %q, want park", s2.TriageDefault)
	}
}

func TestMarker_WriteAndCheck(t *testing.T) {
	dir := t.TempDir()
	s, _ := state.Load(dir, "session-xyz")

	if s.MarkerExists() {
		t.Error("marker should not exist before write")
	}

	if err := s.WriteMarker(); err != nil {
		t.Fatalf("WriteMarker failed: %v", err)
	}
	if !s.MarkerExists() {
		t.Error("marker should exist after write")
	}

	s.DeleteMarker()
	if s.MarkerExists() {
		t.Error("marker should not exist after delete")
	}
}

func TestResetCompactionCount(t *testing.T) {
	dir := t.TempDir()
	s, err := state.Load(dir, "test-reset")
	if err != nil {
		t.Fatal(err)
	}
	s.IncrementCompactionCount()
	s.IncrementCompactionCount()
	if s.CompactionCount != 2 {
		t.Fatalf("expected 2, got %d", s.CompactionCount)
	}
	s.ResetCompactionCount()
	if s.CompactionCount != 0 {
		t.Fatalf("expected 0 after reset, got %d", s.CompactionCount)
	}
	s2, err := state.Load(dir, "test-reset")
	if err != nil {
		t.Fatal(err)
	}
	if s2.CompactionCount != 0 {
		t.Fatalf("expected 0 on reload, got %d", s2.CompactionCount)
	}
}

func TestState_SessionDir(t *testing.T) {
	dir := t.TempDir()
	s, _ := state.Load(dir, "session-abc")
	expected := filepath.Join(dir, "sessions", "session-abc")
	if s.Dir() != expected {
		t.Errorf("Dir: got %q, want %q", s.Dir(), expected)
	}
}
