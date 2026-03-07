package hook

import (
	"os"
	"path/filepath"
	"time"
)

// pruneOldSessions removes session directories older than maxAge.
// Sessions are identified by their directory mtime. The current session is skipped.
func pruneOldSessions(baseDir string, maxAge time.Duration, logPath, currentSessionID string) {
	sessionsDir := filepath.Join(baseDir, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, e := range entries {
		if !e.IsDir() || e.Name() == currentSessionID {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			sessionDir := filepath.Join(sessionsDir, e.Name())
			if err := os.RemoveAll(sessionDir); err == nil {
				logEvent(logPath, currentSessionID, "PRUNE",
					"removed stale session "+e.Name())
			}
		}
	}
}
