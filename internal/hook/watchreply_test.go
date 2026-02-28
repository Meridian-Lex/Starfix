package hook_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/meridian-lex/starfix/internal/hook"
	"github.com/meridian-lex/starfix/internal/state"
)

func TestWatchReply_ReplyReceived(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.TimeoutSeconds = 2
	cfg.TelegramAdmiralID = 121956871

	sentAt := time.Now().UTC()
	s, _ := state.Load(dir, "session-wr-1")
	s.EscalationPending = true
	s.TriageDefault = "continue"
	s.EscalationSentAt = sentAt
	s.Save()

	logPath := filepath.Join(dir, "telegram-inbound.log")
	// Write log entry with timestamp strictly after escalation sent time
	// Add 1 second to ensure RFC3339 formatting produces a later timestamp
	replyTime := sentAt.Add(1100 * time.Millisecond)
	entry := map[string]interface{}{
		"timestamp": replyTime.Format(time.RFC3339),
		"from":      map[string]interface{}{"id": float64(121956871)},
		"text":      "continue",
	}
	b, _ := json.Marshal(entry)
	os.WriteFile(logPath, append(b, '\n'), 0644)
	cfg.TelegramInboundLog = logPath

	hook.RunWatchReply("session-wr-1", cfg, dir)

	s2, _ := state.Load(dir, "session-wr-1")
	if !s2.ReplyReceived {
		t.Error("ReplyReceived should be true after reply found")
	}
	if s2.ReplyText != "continue" {
		t.Errorf("ReplyText: got %q, want continue", s2.ReplyText)
	}
}

func TestWatchReply_Timeout(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.TimeoutSeconds = 1
	cfg.TelegramAdmiralID = 121956871

	logPath := filepath.Join(dir, "empty-inbound.log")
	os.WriteFile(logPath, []byte{}, 0644)
	cfg.TelegramInboundLog = logPath

	s, _ := state.Load(dir, "session-wr-2")
	s.EscalationPending = true
	s.TriageDefault = "park"
	s.EscalationSentAt = time.Now().UTC()
	s.Save()

	hook.RunWatchReply("session-wr-2", cfg, dir)

	s2, _ := state.Load(dir, "session-wr-2")
	if !s2.TimeoutFired {
		t.Error("TimeoutFired should be true after timeout")
	}
	if s2.TimeoutAction != "park" {
		t.Errorf("TimeoutAction: got %q, want park", s2.TimeoutAction)
	}
}
