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
	os.MkdirAll(sessionsDir, 0755)

	// Create an old session directory (simulate old mtime).
	oldDir := filepath.Join(sessionsDir, "old-session-id")
	os.Mkdir(oldDir, 0755)
	oldTime := time.Now().Add(-15 * 24 * time.Hour)
	os.Chtimes(oldDir, oldTime, oldTime)

	// Create a recent session directory.
	recentDir := filepath.Join(sessionsDir, "recent-session-id")
	os.Mkdir(recentDir, 0755)

	// Create "current" session directory (should never be pruned, even if old).
	currentDir := filepath.Join(sessionsDir, "current-session-id")
	os.Mkdir(currentDir, 0755)
	os.Chtimes(currentDir, oldTime, oldTime)

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
