package hook_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
	tests := []struct {
		name  string
		setup func(*testing.T, *config.Config)
	}{
		{
			name: "Interactive",
			setup: func(t *testing.T, cfg *config.Config) {
				// No lock files — interactive mode.
			},
		},
		{
			name: "Ralph",
			setup: func(t *testing.T, cfg *config.Config) {
				writeLock(t, cfg.RalphLockPath)
			},
		},
		{
			name: "Autonomous",
			setup: func(t *testing.T, cfg *config.Config) {
				writeLock(t, cfg.AutonomousLockPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := testConfig(dir)
			input := hookInput("session-test-1")
			tt.setup(t, cfg)

			// Marker should be written regardless of mode.
			hook.HandlePreCompact(input, cfg, dir)

			s, _ := state.Load(dir, "session-test-1")
			if !s.MarkerExists() {
				t.Error("marker file should exist after precompact in any mode")
			}
		})
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

func TestPreCompact_RalphEpochReset(t *testing.T) {
	// Simulates two ralph loops in the same session.
	// After the first loop's lock is removed and a new one written (new mtime),
	// the count should reset to 0 and escalation should clear.
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.TelegramEnabled = false
	input := hookInput("session-epoch")

	// --- First ralph loop: drive count past escalation threshold (8) ---
	writeLock(t, cfg.RalphLockPath)
	for i := 0; i < 9; i++ {
		hook.HandlePreCompact(input, cfg, dir)
	}
	s1, _ := state.Load(dir, "session-epoch")
	if s1.CompactionCount != 9 {
		t.Errorf("after first loop: got count %d, want 9", s1.CompactionCount)
	}
	if !s1.EscalationPending {
		t.Error("EscalationPending should be true after exceeding escalation threshold")
	}

	// --- Simulate loop end + new loop start: recreate lock with newer mtime ---
	// Remove and rewrite the lock file so its mtime is strictly newer.
	// Use 1.1s sleep to cover filesystems with 1-second mtime granularity (e.g., HFS+).
	os.Remove(cfg.RalphLockPath)
	time.Sleep(1100 * time.Millisecond)
	writeLock(t, cfg.RalphLockPath)

	// --- Second ralph loop: first compaction should reset count to 1 ---
	hook.HandlePreCompact(input, cfg, dir)
	s2, _ := state.Load(dir, "session-epoch")
	if s2.CompactionCount != 1 {
		t.Errorf("after epoch reset: got count %d, want 1 (reset + 1 new compaction)", s2.CompactionCount)
	}
	if s2.EscalationPending {
		t.Error("EscalationPending should be cleared after epoch reset")
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

func TestPreCompact_ResetsCount_NewRalphLoop(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	writeLock(t, cfg.RalphLockPath)
	input := hookInput("session-ralph-reset")

	// Run 3 compactions to build up count.
	for i := 0; i < 3; i++ {
		hook.HandlePreCompact(input, cfg, dir)
	}
	s, _ := state.Load(dir, "session-ralph-reset")
	if s.CompactionCount != 3 {
		t.Fatalf("expected CompactionCount 3 before reset, got %d", s.CompactionCount)
	}

	// Simulate a new ralph loop: remove the lock file and create a fresh one.
	// The new file gets a different inode, producing a different epoch token
	// regardless of filesystem mtime granularity.
	os.Remove(cfg.RalphLockPath)
	writeLock(t, cfg.RalphLockPath)

	// Next compaction should reset before incrementing, resulting in count 1.
	hook.HandlePreCompact(input, cfg, dir)
	s2, _ := state.Load(dir, "session-ralph-reset")
	if s2.CompactionCount != 1 {
		t.Errorf("expected CompactionCount 1 after new loop reset, got %d", s2.CompactionCount)
	}
}

func TestPreCompact_ResetsCount_NewAutonomousLoop(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	writeLock(t, cfg.AutonomousLockPath)
	input := hookInput("session-auto-reset")

	// Run 3 compactions to build up count.
	for i := 0; i < 3; i++ {
		hook.HandlePreCompact(input, cfg, dir)
	}
	s, _ := state.Load(dir, "session-auto-reset")
	if s.CompactionCount != 3 {
		t.Fatalf("expected CompactionCount 3 before reset, got %d", s.CompactionCount)
	}

	// Simulate a new autonomous loop by touching the lock file.
	stateFile := s.StateFile()
	past := time.Now().Add(-10 * time.Second)
	os.Chtimes(stateFile, past, past)
	writeLock(t, cfg.AutonomousLockPath)

	// Next compaction should reset before incrementing, resulting in count 1.
	hook.HandlePreCompact(input, cfg, dir)
	s2, _ := state.Load(dir, "session-auto-reset")
	if s2.CompactionCount != 1 {
		t.Errorf("expected CompactionCount 1 after new loop reset, got %d", s2.CompactionCount)
	}
}

func TestPreCompact_NoReset_OldLockFile(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	writeLock(t, cfg.RalphLockPath)
	input := hookInput("session-ralph-old-lock")

	// Run 3 compactions.
	for i := 0; i < 3; i++ {
		hook.HandlePreCompact(input, cfg, dir)
	}
	s, _ := state.Load(dir, "session-ralph-old-lock")
	if s.CompactionCount != 3 {
		t.Fatalf("expected CompactionCount 3 before check, got %d", s.CompactionCount)
	}

	// The lock file is the same file (same inode, same mtime) — epoch token
	// is unchanged so no reset should occur. Simply proceed to the next compaction.

	// Next compaction should just increment to 4, no reset.
	hook.HandlePreCompact(input, cfg, dir)
	s2, _ := state.Load(dir, "session-ralph-old-lock")
	if s2.CompactionCount != 4 {
		t.Errorf("expected CompactionCount 4 (no reset for unchanged lock), got %d", s2.CompactionCount)
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

func TestPreCompact_RalphEpochReset_WithBothLocks(t *testing.T) {
	// When both ralph and autonomous locks are present, a ralph epoch reset
	// should still clear state correctly (ralph takes precedence).
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.TelegramEnabled = false
	input := hookInput("session-both-epoch")

	writeLock(t, cfg.RalphLockPath)
	writeLock(t, cfg.AutonomousLockPath)

	// Build up compaction count under ralph+autonomous.
	for i := 0; i < 3; i++ {
		hook.HandlePreCompact(input, cfg, dir)
	}
	s1, _ := state.Load(dir, "session-both-epoch")
	if s1.CompactionCount != 3 {
		t.Errorf("before epoch reset: got count %d, want 3", s1.CompactionCount)
	}

	// Recreate ralph lock (new epoch) while autonomous lock remains.
	os.Remove(cfg.RalphLockPath)
	time.Sleep(1100 * time.Millisecond)
	writeLock(t, cfg.RalphLockPath)

	hook.HandlePreCompact(input, cfg, dir)
	s2, _ := state.Load(dir, "session-both-epoch")
	if s2.CompactionCount != 1 {
		t.Errorf("after epoch reset with both locks: got count %d, want 1", s2.CompactionCount)
	}
}

func TestPreCompact_AutonomousToRalphTransition(t *testing.T) {
	// When compactions accumulate in autonomous mode and then ralph starts,
	// the count should reset (ralph gets a clean slate) and the reset should
	// be logged as a cross-mode transition.
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.TelegramEnabled = false
	input := hookInput("session-cross-mode")

	// --- Autonomous phase: accumulate 5 compactions ---
	writeLock(t, cfg.AutonomousLockPath)
	for i := 0; i < 5; i++ {
		hook.HandlePreCompact(input, cfg, dir)
	}
	s1, _ := state.Load(dir, "session-cross-mode")
	if s1.CompactionCount != 5 {
		t.Fatalf("autonomous phase: got count %d, want 5", s1.CompactionCount)
	}

	// --- Transition to ralph: create ralph lock (autonomous lock remains) ---
	writeLock(t, cfg.RalphLockPath)
	hook.HandlePreCompact(input, cfg, dir)
	s2, _ := state.Load(dir, "session-cross-mode")

	// Ralph epoch reset should have cleared the autonomous count and started fresh.
	if s2.CompactionCount != 1 {
		t.Errorf("after autonomous-to-ralph transition: got count %d, want 1 (reset + 1 new)", s2.CompactionCount)
	}
	if s2.EscalationPending {
		t.Error("EscalationPending should be false after cross-mode reset")
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

func TestPreCompact_ZeroThresholdFallsBackToGlobal(t *testing.T) {
	// When a mode-specific threshold is zero (unset), the global threshold
	// should be used as fallback. Verify this by setting ralph thresholds to 0
	// and confirming escalation fires at the global threshold.
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.RalphSummaryThreshold = 0    // should fall back to cfg.SummaryThreshold (2)
	cfg.RalphEscalationThreshold = 0 // should fall back to cfg.EscalationThreshold (3)
	cfg.TelegramEnabled = false
	writeLock(t, cfg.RalphLockPath)
	input := hookInput("session-zero-threshold")

	// 3 compactions should hit global escalation threshold (3).
	for i := 0; i < 3; i++ {
		hook.HandlePreCompact(input, cfg, dir)
	}

	s, _ := state.Load(dir, "session-zero-threshold")
	if s.CompactionCount != 3 {
		t.Errorf("CompactionCount: got %d, want 3", s.CompactionCount)
	}
	if !s.EscalationPending {
		t.Error("EscalationPending should be true — global fallback threshold (3) reached with zero ralph threshold")
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
		StatePath:                     filepath.Join(dir, "STATE.md"),
		MemoryPath:                    filepath.Join(dir, "MEMORY.md"),
	}
}
