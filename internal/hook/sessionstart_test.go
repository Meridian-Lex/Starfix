package hook_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

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

func TestSessionStart_StaleMarker_SkipsInjection(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ss-stale")

	s, _ := state.Load(dir, "session-ss-stale")
	s.WriteMarker()

	// Backdate the marker by 5 hours to make it stale (threshold is 4h).
	past := time.Now().Add(-5 * time.Hour)
	os.Chtimes(s.MarkerFile(), past, past)

	output := hook.HandleSessionStart(input, cfg, dir)
	if output != "" {
		t.Errorf("stale marker should produce empty output, got: %q", output)
	}

	// Marker should be deleted.
	s2, _ := state.Load(dir, "session-ss-stale")
	if s2.MarkerExists() {
		t.Error("stale marker should be deleted by sessionstart")
	}
}

func TestApplyPendingSignals_ReplyReceived(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ss-reply")

	s, _ := state.Load(dir, "session-ss-reply")
	s.ReplyReceived = true
	s.ReplyText = "continue please"
	s.EscalationPending = true
	s.WriteMarker()
	s.Save()

	output := hook.HandleSessionStart(input, cfg, dir)
	if !strings.Contains(output, "continue please") {
		t.Errorf("output should contain reply text, got: %q", output)
	}
	if !strings.Contains(output, "ADMIRAL REPLY") {
		t.Errorf("output should contain ADMIRAL REPLY header, got: %q", output)
	}
}

func TestApplyPendingSignals_TimeoutFired_Park(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ss-timeout")

	s, _ := state.Load(dir, "session-ss-timeout")
	s.TimeoutFired = true
	s.TimeoutAction = "park"
	s.EscalationPending = true
	s.WriteMarker()
	s.Save()

	output := hook.HandleSessionStart(input, cfg, dir)
	if !strings.Contains(output, "STARFIX DIRECTIVE") {
		t.Errorf("output should contain STARFIX DIRECTIVE for park timeout, got: %q", output)
	}
}

func TestApplyPendingSignals_TimeoutFired_Continue(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ss-timeout-cont")

	s, _ := state.Load(dir, "session-ss-timeout-cont")
	s.TimeoutFired = true
	s.TimeoutAction = "continue"
	s.WriteMarker()
	s.Save()

	output := hook.HandleSessionStart(input, cfg, dir)
	// No STARFIX DIRECTIVE for continue action, but output should still have STARFIX header.
	if !strings.Contains(output, "STARFIX") {
		t.Errorf("output should contain STARFIX header, got: %q", output)
	}
	if strings.Contains(output, "STARFIX DIRECTIVE") {
		t.Errorf("output should not contain STARFIX DIRECTIVE for continue timeout, got: %q", output)
	}
}

func TestReadInput_ValidJSON(t *testing.T) {
	data := []byte(`{"session_id":"abc-123","cwd":"/home/test"}`)
	input, err := hook.ReadInput(data)
	if err != nil {
		t.Fatalf("ReadInput failed: %v", err)
	}
	if input.SessionID != "abc-123" {
		t.Errorf("SessionID: got %q, want abc-123", input.SessionID)
	}
	if input.CWD != "/home/test" {
		t.Errorf("CWD: got %q, want /home/test", input.CWD)
	}
}

func TestReadInput_InvalidJSON(t *testing.T) {
	_, err := hook.ReadInput([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestReadInput_EmptyObject(t *testing.T) {
	input, err := hook.ReadInput([]byte(`{}`))
	if err != nil {
		t.Fatalf("ReadInput failed on empty object: %v", err)
	}
	if input.SessionID != "" {
		t.Errorf("SessionID should be empty for empty object, got %q", input.SessionID)
	}
}
