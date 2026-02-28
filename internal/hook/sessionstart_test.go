package hook_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/meridian-lex/starfix/internal/hook"
	"github.com/meridian-lex/starfix/internal/state"
)

func TestSessionStart_NoMarker_EmptyOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)

	output := hook.HandleSessionStart(hookInput("session-ss-1"), cfg, dir)
	if output != "" {
		t.Errorf("expected empty output when no marker, got: %q", output)
	}
}

func TestSessionStart_WithMarker_InjectsContext(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ss-2")

	s, _ := state.Load(dir, "session-ss-2")
	s.WriteMarker()

	output := hook.HandleSessionStart(input, cfg, dir)

	if !strings.Contains(output, "STARFIX") {
		t.Errorf("output should contain STARFIX header, got: %q", output)
	}
}

func TestSessionStart_WithMarker_DeletesMarker(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ss-3")

	s, _ := state.Load(dir, "session-ss-3")
	s.WriteMarker()

	hook.HandleSessionStart(input, cfg, dir)

	s2, _ := state.Load(dir, "session-ss-3")
	if s2.MarkerExists() {
		t.Error("marker should be deleted after sessionstart")
	}
}

func TestSessionStart_OutputIsValidJSON(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ss-4")

	s, _ := state.Load(dir, "session-ss-4")
	s.WriteMarker()

	output := hook.HandleSessionStart(input, cfg, dir)
	if output == "" {
		t.Fatal("expected non-empty JSON output")
	}
	var v interface{}
	if err := json.Unmarshal([]byte(output), &v); err != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", err, output)
	}
}
