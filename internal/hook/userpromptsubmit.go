package hook

import (
	"fmt"

	starfixctx "github.com/meridian-lex/starfix/internal/context"
	"github.com/meridian-lex/starfix/internal/config"
	"github.com/meridian-lex/starfix/internal/state"
)

// HandleUserPromptSubmit processes the UserPromptSubmit hook event.
// Acts as fallback if SessionStart did not fire, and handles pending reply/timeout flags.
// Returns plain text to inject into context (stdout), or empty string.
func HandleUserPromptSubmit(input Input, cfg *config.Config, baseDir string) string {
	s, err := state.Load(baseDir, input.SessionID)
	if err != nil {
		return ""
	}

	var payload string

	if s.ReplyReceived {
		payload += fmt.Sprintf("\n--- ADMIRAL REPLY ---\nFleet Admiral replied: %s\n", s.ReplyText)
		s.ReplyReceived = false
		s.ReplyText = ""
		s.EscalationPending = false
		s.Save()
	} else if s.TimeoutFired {
		if s.TimeoutAction == "park" {
			payload += "\n--- STARFIX DIRECTIVE ---\nNo Admiral reply received within timeout. Triage recommended PARK. Please wrap up current work and stop.\n"
		}
		s.TimeoutFired = false
		s.EscalationPending = false
		s.Save()
	}

	if s.MarkerExists() {
		corePayload := starfixctx.BuildCore(cfg)
		if cfg.ProjectContext && input.CWD != "" {
			if projectLayer := starfixctx.BuildProject(input.CWD); projectLayer != "" {
				corePayload += "\n" + projectLayer
			}
		}
		payload = corePayload + payload
		s.DeleteMarker()
		s.Save()
		logEvent(cfg.LogPath, input.SessionID, "INJECT", "context injected via userpromptsubmit fallback")
	}

	return payload
}
