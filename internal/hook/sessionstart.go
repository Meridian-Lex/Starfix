package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	starfixctx "github.com/meridian-lex/starfix/internal/context"
	"github.com/meridian-lex/starfix/internal/config"
	"github.com/meridian-lex/starfix/internal/state"
)

type sessionStartOutput struct {
	HookSpecificOutput sessionStartSpecific `json:"hookSpecificOutput"`
}

type sessionStartSpecific struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// isStaleMarker returns true if the marker file is older than the given max age.
func isStaleMarker(markerPath string, maxAge time.Duration) bool {
	info, err := os.Stat(markerPath)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) > maxAge
}

// buildPayload constructs the context injection payload from config and escalation state.
func buildPayload(s *state.SessionState, cfg *config.Config, cwd string) string {
	payload := starfixctx.BuildCore(cfg)
	if cfg.ProjectContext && cwd != "" {
		if projectLayer := starfixctx.BuildProject(cwd); projectLayer != "" {
			payload += "\n" + projectLayer
		}
	}

	if s.ReplyReceived {
		payload += fmt.Sprintf("\n--- ADMIRAL REPLY ---\nFleet Admiral replied: %s\n", s.ReplyText)
		s.ReplyReceived = false
		s.ReplyText = ""
		s.EscalationPending = false
	} else if s.TimeoutFired {
		if s.TimeoutAction == "park" {
			payload += "\n--- STARFIX DIRECTIVE ---\nNo Admiral reply received within timeout. Triage recommended PARK. Please wrap up current work and stop.\n"
		}
		s.TimeoutFired = false
		s.EscalationPending = false
	}

	return payload
}

// cleanupWatchReply terminates any previously-spawned watch-reply process and
// removes its PID file.
func cleanupWatchReply(sessionDir string) {
	pidFile := filepath.Join(sessionDir, "watch-reply.pid")
	killExistingWatchReply(pidFile)
	os.Remove(pidFile)
}

// formatOutput marshals the sessionstart hook output as JSON.
func formatOutput(payload string) string {
	out := sessionStartOutput{
		HookSpecificOutput: sessionStartSpecific{
			HookEventName:     "SessionStart",
			AdditionalContext: payload,
		},
	}
	b, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(b)
}

// HandleSessionStart processes the SessionStart hook event.
// Returns JSON string for additionalContext injection, or empty string if no action needed.
func HandleSessionStart(input Input, cfg *config.Config, baseDir string) string {
	pruneOldSessions(baseDir, 14*24*time.Hour, cfg.LogPath, input.SessionID)

	s, err := state.Load(baseDir, input.SessionID)
	if err != nil {
		logEvent(cfg.LogPath, input.SessionID, "ERROR", fmt.Sprintf("sessionstart load state: %v", err))
		return ""
	}

	if !s.MarkerExists() {
		return ""
	}

	// Check for stale marker (e.g. from a crashed SessionStart).
	if isStaleMarker(s.MarkerFile(), 4*time.Hour) {
		logEvent(cfg.LogPath, input.SessionID, "WARN",
			"compact-pending marker is stale (>4h) — deleting without injection")
		s.DeleteMarker()
		if err := s.Save(); err != nil {
			logEvent(cfg.LogPath, input.SessionID, "ERROR",
				fmt.Sprintf("failed to save state after stale marker cleanup: %v", err))
		}
		return ""
	}

	payload := buildPayload(s, cfg, input.CWD)

	s.DeleteMarker()
	if err := s.Save(); err != nil {
		logEvent(cfg.LogPath, input.SessionID, "ERROR",
			fmt.Sprintf("failed to save state after context injection: %v", err))
	}

	cleanupWatchReply(s.Dir())

	logEvent(cfg.LogPath, input.SessionID, "INJECT", "context injected via sessionstart")

	return formatOutput(payload)
}
