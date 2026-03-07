package hook_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/meridian-lex/starfix/internal/config"
	"github.com/meridian-lex/starfix/internal/hook"
	"github.com/meridian-lex/starfix/internal/state"
)

// writeLock creates a lock file at the given path.
func writeLock(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("lock"), 0644); err != nil {
		t.Fatalf("writeLock: %v", err)
	}
}

func TestPreCompact_WritesMarker(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-test-1")

	// Marker should be written regardless of mode.
	hook.HandlePreCompact(input, cfg, dir)

	s, _ := state.Load(dir, "session-test-1")
	if !s.MarkerExists() {
		t.Error("marker file should exist after precompact in any mode")
	}
}

func TestPreCompact_NoCountInInteractiveMode(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-interactive")

	// No lock files — interactive mode.
	hook.HandlePreCompact(input, cfg, dir)
	hook.HandlePreCompact(input, cfg, dir)

	s, _ := state.Load(dir, "session-interactive")
	if s.CompactionCount != 0 {
		t.Errorf("CompactionCount should remain 0 in interactive mode, got %d", s.CompactionCount)
	}
	if s.EscalationPending {
		t.Error("EscalationPending should not be set in interactive mode")
	}
}

func TestPreCompact_IncrementsCount_RalphMode(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	writeLock(t, cfg.RalphLockPath)
	input := hookInput("session-ralph")

	hook.HandlePreCompact(input, cfg, dir)
	hook.HandlePreCompact(input, cfg, dir)

	s, _ := state.Load(dir, "session-ralph")
	if s.CompactionCount != 2 {
		t.Errorf("CompactionCount: got %d, want 2", s.CompactionCount)
	}
}

func TestPreCompact_IncrementsCount_AutonomousMode(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	writeLock(t, cfg.AutonomousLockPath)
	input := hookInput("session-autonomous")

	hook.HandlePreCompact(input, cfg, dir)
	hook.HandlePreCompact(input, cfg, dir)

	s, _ := state.Load(dir, "session-autonomous")
	if s.CompactionCount != 2 {
		t.Errorf("CompactionCount: got %d, want 2", s.CompactionCount)
	}
}

func TestPreCompact_RalphTakesPrecedenceOverAutonomous(t *testing.T) {
	// When both locks are present, ralph thresholds (tighter) apply.
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.RalphSummaryThreshold = 2
	cfg.RalphEscalationThreshold = 4
	cfg.AutonomousSummaryThreshold = 10
	cfg.AutonomousEscalationThreshold = 20
	cfg.TelegramEnabled = false
	writeLock(t, cfg.RalphLockPath)
	writeLock(t, cfg.AutonomousLockPath)
	input := hookInput("session-both")

	// 4 compactions — should hit ralph escalation threshold (4), not autonomous (20).
	for i := 0; i < 4; i++ {
		hook.HandlePreCompact(input, cfg, dir)
	}

	s, _ := state.Load(dir, "session-both")
	if !s.EscalationPending {
		t.Error("EscalationPending should be true — ralph escalation threshold reached")
	}
}

func TestPreCompact_SetsEscalationAtThreshold(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.RalphEscalationThreshold = 2
	cfg.TelegramEnabled = false
	writeLock(t, cfg.RalphLockPath)
	input := hookInput("session-escalation")

	// First call: CompactionCount becomes 1 (below threshold).
	hook.HandlePreCompact(input, cfg, dir)
	s1, _ := state.Load(dir, "session-escalation")
	if s1.EscalationPending {
		t.Error("EscalationPending should be false at count 1")
	}

	// Second call: CompactionCount becomes 2 (at threshold), triggers escalation.
	hook.HandlePreCompact(input, cfg, dir)
	s2, _ := state.Load(dir, "session-escalation")
	if !s2.EscalationPending {
		t.Error("EscalationPending should be true at threshold")
	}
	if s2.TriageDefault != "continue" && s2.TriageDefault != "park" {
		t.Errorf("TriageDefault should be set, got %q", s2.TriageDefault)
	}
}

func hookInput(sessionID string) hook.Input {
	return hook.Input{SessionID: sessionID, CWD: "/tmp"}
}

func testConfig(dir string) *config.Config {
	return &config.Config{
		SummaryThreshold:              2,
		EscalationThreshold:           3,
		RalphSummaryThreshold:         4,
		RalphEscalationThreshold:      8,
		AutonomousSummaryThreshold:    6,
		AutonomousEscalationThreshold: 12,
		AutonomousLockPath:            filepath.Join(dir, "AUTONOMOUS-MODE.lock"),
		RalphLockPath:                 filepath.Join(dir, "RALPH-LOOP.lock"),
		TimeoutSeconds:                5,
		TelegramEnabled:               false,
		LogPath:                       filepath.Join(dir, "starfix.log"),
		TaskQueuePath:                 filepath.Join(dir, "TASK-QUEUE.md"),
	}
}
