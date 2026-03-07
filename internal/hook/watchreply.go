package hook

import (
	"fmt"
	"time"

	"github.com/meridian-lex/starfix/internal/config"
	"github.com/meridian-lex/starfix/internal/state"
	"github.com/meridian-lex/starfix/internal/telegram"
)

// RunWatchReply polls for an Admiral reply until timeout, then executes triage default.
// Intended to run as a background process spawned by HandlePreCompact.
func RunWatchReply(sessionID string, cfg *config.Config, baseDir string) {
	s, err := state.Load(baseDir, sessionID)
	if err != nil {
		return
	}
	if !s.EscalationPending {
		return
	}

	deadline := s.EscalationSentAt.Add(time.Duration(cfg.TimeoutSeconds) * time.Second)
	pollInterval := 5 * time.Second

	// Adapt poll interval to deadline — use min(5s, deadline/2)
	if remaining := time.Until(deadline); remaining < 10*time.Second {
		pollInterval = remaining / 2
		if pollInterval < 100*time.Millisecond {
			pollInterval = 100 * time.Millisecond
		}
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	deadlineCh := time.After(time.Until(deadline))

	for {
		select {
		case <-deadlineCh:
			// Reload state before acting — ResetLoop may have cleared EscalationPending.
			if fresh, err := state.Load(baseDir, sessionID); err == nil {
				s = fresh
			}
			if !s.EscalationPending {
				return
			}
			// Timeout path
			s.TimeoutFired = true
			s.TimeoutAction = s.TriageDefault
			s.EscalationPending = false
			s.Save()

			if cfg.TelegramEnabled {
				msg := fmt.Sprintf("[Starfix] No reply — proceeding to %s\nSession: %s",
					s.TriageDefault, shortID(sessionID))
				telegram.Send(cfg.TelegramBinary, msg)
			}
			return

		case <-ticker.C:
			// Reload state each tick — ResetLoop may have cleared EscalationPending.
			if fresh, err := state.Load(baseDir, sessionID); err == nil {
				s = fresh
			}
			if !s.EscalationPending {
				return
			}
			reply, found := telegram.CheckInbound(
				cfg.TelegramInboundLog,
				s.EscalationSentAt,
				cfg.TelegramAdmiralID,
			)
			if found {
				s.ReplyReceived = true
				s.ReplyText = reply
				s.EscalationPending = false
				s.Save()

				if cfg.TelegramEnabled {
					msg := fmt.Sprintf("[Starfix] Reply received from Admiral — %s — session %s",
						reply, shortID(sessionID))
					telegram.Send(cfg.TelegramBinary, msg)
				}
				return
			}
		}
	}
}
