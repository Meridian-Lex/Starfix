package hook_test

import (
	"strings"
	"testing"

	"github.com/meridian-lex/starfix/internal/hook"
	"github.com/meridian-lex/starfix/internal/state"
)

func TestUserPromptSubmit_NoMarker_EmptyOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)

	output := hook.HandleUserPromptSubmit(hookInput("session-ups-1"), cfg, dir)
	if output != "" {
		t.Errorf("expected empty output when no marker, got: %q", output)
	}
}

func TestUserPromptSubmit_WithMarker_InjectsContext(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ups-2")

	s, _ := state.Load(dir, "session-ups-2")
	s.WriteMarker()

	output := hook.HandleUserPromptSubmit(input, cfg, dir)
	if !strings.Contains(output, "STARFIX") {
		t.Errorf("expected STARFIX header in output, got: %q", output)
	}
}

func TestUserPromptSubmit_WithMarker_DeletesMarker(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ups-3")

	s, _ := state.Load(dir, "session-ups-3")
	s.WriteMarker()

	hook.HandleUserPromptSubmit(input, cfg, dir)

	s2, _ := state.Load(dir, "session-ups-3")
	if s2.MarkerExists() {
		t.Error("marker should be deleted after userpromptsubmit")
	}
}

func TestUserPromptSubmit_ParkFlag_InjectsDirective(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ups-4")

	s, _ := state.Load(dir, "session-ups-4")
	s.TimeoutFired = true
	s.TimeoutAction = "park"
	s.Save()

	output := hook.HandleUserPromptSubmit(input, cfg, dir)
	if !strings.Contains(output, "park") && !strings.Contains(output, "PARK") {
		t.Errorf("expected park directive in output, got: %q", output)
	}
}
