package hook

import (
	"fmt"
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
		logEvent(logPath, currentSessionID, "PRUNE_FAIL",
			fmt.Sprintf("failed to read sessions directory %s: %v", sessionsDir, err))
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
			if err := os.RemoveAll(sessionDir); err != nil {
				logEvent(logPath, currentSessionID, "PRUNE_FAIL",
					fmt.Sprintf("failed to remove session %s: %v", e.Name(), err))
			} else {
				logEvent(logPath, currentSessionID, "PRUNE",
					fmt.Sprintf("removed session %s", e.Name()))
			}
		}
	}
}
