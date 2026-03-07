package hook

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPruneOldSessions(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatalf("setup: MkdirAll: %v", err)
	}

	// Create an old session directory (simulate old mtime).
	oldDir := filepath.Join(sessionsDir, "old-session-id")
	if err := os.Mkdir(oldDir, 0755); err != nil {
		t.Fatalf("setup: Mkdir old: %v", err)
	}
	oldTime := time.Now().Add(-15 * 24 * time.Hour)
	if err := os.Chtimes(oldDir, oldTime, oldTime); err != nil {
		t.Fatalf("setup: Chtimes old: %v", err)
	}

	// Create a recent session directory.
	recentDir := filepath.Join(sessionsDir, "recent-session-id")
	if err := os.Mkdir(recentDir, 0755); err != nil {
		t.Fatalf("setup: Mkdir recent: %v", err)
	}

	// Create "current" session directory (should never be pruned, even if old).
	currentDir := filepath.Join(sessionsDir, "current-session-id")
	if err := os.Mkdir(currentDir, 0755); err != nil {
		t.Fatalf("setup: Mkdir current: %v", err)
	}
	if err := os.Chtimes(currentDir, oldTime, oldTime); err != nil {
		t.Fatalf("setup: Chtimes current: %v", err)
	}

	pruneOldSessions(dir, 14*24*time.Hour, filepath.Join(dir, "starfix.log"), "current-session-id")

	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("old session should have been pruned")
	}
	if _, err := os.Stat(recentDir); os.IsNotExist(err) {
		t.Error("recent session should not have been pruned")
	}
	if _, err := os.Stat(currentDir); os.IsNotExist(err) {
		t.Error("current session should never be pruned")
	}
}
