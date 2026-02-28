package hook_test

import (
	"path/filepath"
	"testing"

	"github.com/meridian-lex/starfix/internal/config"
	"github.com/meridian-lex/starfix/internal/hook"
	"github.com/meridian-lex/starfix/internal/state"
)

func TestPreCompact_WritesMarker(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-test-1")

	hook.HandlePreCompact(input, cfg, dir)

	s, _ := state.Load(dir, "session-test-1")
	if !s.MarkerExists() {
		t.Error("marker file should exist after precompact")
	}
}

func TestPreCompact_IncrementsCount(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-test-2")

	hook.HandlePreCompact(input, cfg, dir)
	hook.HandlePreCompact(input, cfg, dir)

	s, _ := state.Load(dir, "session-test-2")
	if s.CompactionCount != 2 {
		t.Errorf("CompactionCount: got %d, want 2", s.CompactionCount)
	}
}

func TestPreCompact_SetsEscalationAtThreshold(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.EscalationThreshold = 2
	cfg.TelegramEnabled = false
	input := hookInput("session-test-3")

	// First call: CompactionCount becomes 1 (below threshold)
	hook.HandlePreCompact(input, cfg, dir)
	s1, _ := state.Load(dir, "session-test-3")
	if s1.EscalationPending {
		t.Error("EscalationPending should be false at count 1")
	}

	// Second call: CompactionCount becomes 2 (at threshold), triggers escalation
	hook.HandlePreCompact(input, cfg, dir)
	s2, _ := state.Load(dir, "session-test-3")
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
		SummaryThreshold:    2,
		EscalationThreshold: 3,
		TimeoutSeconds:      5,
		TelegramEnabled:     false,
		LogPath:             filepath.Join(dir, "starfix.log"),
		TaskQueuePath:       filepath.Join(dir, "TASK-QUEUE.md"),
	}
}
