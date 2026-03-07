package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	markerInfo, err := os.Stat(s.MarkerFile())
	if err == nil && time.Since(markerInfo.ModTime()) > 4*time.Hour {
		logEvent(cfg.LogPath, input.SessionID, "WARN",
			"compact-pending marker is stale (>4h) — deleting without injection")
		s.DeleteMarker()
		if err := s.Save(); err != nil {
			logEvent(cfg.LogPath, input.SessionID, "ERROR",
				fmt.Sprintf("failed to save state after stale marker cleanup: %v", err))
		}
		return ""
	}

	payload := starfixctx.BuildCore(cfg)
	if cfg.ProjectContext && input.CWD != "" {
		if projectLayer := starfixctx.BuildProject(input.CWD); projectLayer != "" {
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

	s.DeleteMarker()
	s.Save()

	// Kill any running watch-reply process before removing the PID file.
	pidFile := filepath.Join(s.Dir(), "watch-reply.pid")
	if data, err := os.ReadFile(pidFile); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				if err := proc.Signal(os.Interrupt); err != nil {
					// SIGINT failed; try SIGKILL.
					proc.Kill()
				}
			}
		}
	}
	os.Remove(pidFile)

	logEvent(cfg.LogPath, input.SessionID, "INJECT", "context injected via sessionstart")

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
