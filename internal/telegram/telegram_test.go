package telegram_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/meridian-lex/starfix/internal/telegram"
)

func TestParseInboundLog_FindsReply(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "telegram-inbound.log")

	since := time.Now().Add(-time.Hour)

	entries := []map[string]interface{}{
		{"timestamp": time.Now().UTC().Format(time.RFC3339), "from": map[string]interface{}{"id": float64(121956871)}, "text": "continue"},
	}
	var lines []byte
	for _, e := range entries {
		b, _ := json.Marshal(e)
		lines = append(lines, b...)
		lines = append(lines, '\n')
	}
	os.WriteFile(logPath, lines, 0644)

	reply, found := telegram.CheckInbound(logPath, since, 121956871)
	if !found {
		t.Fatal("expected reply to be found")
	}
	if reply != "continue" {
		t.Errorf("reply: got %q, want continue", reply)
	}
}

func TestParseInboundLog_BeforeSince_NotFound(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "telegram-inbound.log")

	since := time.Now().Add(time.Hour)

	entry := map[string]interface{}{
		"timestamp": time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
		"from":      map[string]interface{}{"id": float64(121956871)},
		"text":      "park",
	}
	b, _ := json.Marshal(entry)
	os.WriteFile(logPath, append(b, '\n'), 0644)

	_, found := telegram.CheckInbound(logPath, since, 121956871)
	if found {
		t.Fatal("should not find reply before since timestamp")
	}
}

func TestParseInboundLog_WrongSender(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "telegram-inbound.log")

	since := time.Now().Add(-time.Hour)
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"from":      map[string]interface{}{"id": float64(999999)},
		"text":      "continue",
	}
	b, _ := json.Marshal(entry)
	os.WriteFile(logPath, append(b, '\n'), 0644)

	_, found := telegram.CheckInbound(logPath, since, 121956871)
	if found {
		t.Fatal("should not find reply from wrong sender")
	}
}

func TestParseInboundLog_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "empty.log")
	os.WriteFile(logPath, []byte{}, 0644)

	_, found := telegram.CheckInbound(logPath, time.Now().Add(-time.Hour), 121956871)
	if found {
		t.Fatal("should not find reply in empty file")
	}
}

func TestSend_CallsBinary(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "send-out.txt")
	fakeBin := filepath.Join(dir, "fake-notify")
	script := "#!/bin/sh\necho \"$@\" > " + outFile + "\n"
	os.WriteFile(fakeBin, []byte(script), 0755)

	err := telegram.Send(fakeBin, "hello fleet")
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("fake binary was not called: %v", err)
	}
	if string(data) != "hello fleet\n" {
		t.Errorf("binary output: got %q, want %q", string(data), "hello fleet\n")
	}
}

func TestSend_ErrorOnMissingBinary(t *testing.T) {
	err := telegram.Send("/nonexistent/binary", "msg")
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
}
