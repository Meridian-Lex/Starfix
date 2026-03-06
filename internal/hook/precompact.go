package hook

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/meridian-lex/starfix/internal/config"
	"github.com/meridian-lex/starfix/internal/state"
	"github.com/meridian-lex/starfix/internal/telegram"
	"github.com/meridian-lex/starfix/internal/triage"
)

// operationalMode describes which autonomous context is active.
type operationalMode int

const (
	modeInteractive operationalMode = iota // no autonomous operation — user is present
	modeAutonomous                         // AUTONOMOUS-MODE.lock active
	modeRalph                              // RALPH-LOOP.lock active (also covers ralph-within-autonomous)
)

// activeMode checks lock files and returns the current operational mode.
// Ralph takes precedence over autonomous when both are active (tighter thresholds).
func activeMode(cfg *config.Config) operationalMode {
	ralphActive := fileExists(cfg.RalphLockPath)
	autonomousActive := fileExists(cfg.AutonomousLockPath)

	switch {
	case ralphActive:
		return modeRalph
	case autonomousActive:
		return modeAutonomous
	default:
		return modeInteractive
	}
}

// thresholds returns (summary, escalation) counts for the given mode.
func thresholds(mode operationalMode, cfg *config.Config) (summary, escalation int) {
	switch mode {
	case modeRalph:
		return cfg.RalphSummaryThreshold, cfg.RalphEscalationThreshold
	case modeAutonomous:
		return cfg.AutonomousSummaryThreshold, cfg.AutonomousEscalationThreshold
	default:
		// fallback — not used in interactive mode but kept for completeness
		return cfg.SummaryThreshold, cfg.EscalationThreshold
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// HandlePreCompact processes the PreCompact hook event.
//
// Context restoration (compact-pending marker) always runs regardless of mode.
// Compaction counting and escalation only activate during ralph loop or autonomous mode —
// compactions in a normal interactive session are handled silently.
func HandlePreCompact(input Input, cfg *config.Config, baseDir string) {
	s, err := state.Load(baseDir, input.SessionID)
	if err != nil {
		logEvent(cfg.LogPath, input.SessionID, "ERROR", fmt.Sprintf("load state: %v", err))
		return
	}

	// Always write the context-restore marker so SessionStart can re-inject state.
	if err := s.WriteMarker(); err != nil {
		logEvent(cfg.LogPath, input.SessionID, "ERROR", fmt.Sprintf("write marker: %v", err))
	}

	mode := activeMode(cfg)

	if mode == modeInteractive {
		// No autonomous operation — log and exit. No counting, no Telegram, no escalation.
		logEvent(cfg.LogPath, input.SessionID, "COMPACT", "compaction (interactive — context marker written)")
		s.Save()
		return
	}

	s.CompactionCount++
	if err := s.Save(); err != nil {
		logEvent(cfg.LogPath, input.SessionID, "ERROR", fmt.Sprintf("save state: %v", err))
	}

	modeLabel := "autonomous"
	if mode == modeRalph {
		modeLabel = "ralph"
	}
	logEvent(cfg.LogPath, input.SessionID, "COMPACT",
		fmt.Sprintf("compaction #%d (%s)", s.CompactionCount, modeLabel))

	summaryThreshold, escalationThreshold := thresholds(mode, cfg)

	if s.CompactionCount >= summaryThreshold && cfg.TelegramEnabled {
		msg := fmt.Sprintf("[Starfix] Compaction #%d (%s) — session %s\nTimestamp: %s",
			s.CompactionCount, modeLabel, shortID(input.SessionID),
			time.Now().UTC().Format(time.RFC3339))
		telegram.Send(cfg.TelegramBinary, msg)
	}

	if s.CompactionCount >= escalationThreshold && !s.EscalationPending {
		taskContent, _ := os.ReadFile(cfg.TaskQueuePath)
		result := triage.Assess(triage.Input{
			CompactionCount:  s.CompactionCount,
			TaskQueueContent: string(taskContent),
		})

		s.EscalationPending = true
		s.TriageDefault = result.Action
		s.EscalationSentAt = time.Now().UTC()
		s.Save()

		if cfg.TelegramEnabled {
			msg := fmt.Sprintf(
				"[Starfix] Context pressure — session %s\nMode: %s | Compaction #%d\nTriage: %s\nRecommended: %s\nWill %s in %ds — reply to override.",
				shortID(input.SessionID), modeLabel, s.CompactionCount,
				result.Reason, result.Action, result.Action, cfg.TimeoutSeconds)
			telegram.Send(cfg.TelegramBinary, msg)

			self, _ := os.Executable()
			cmd := exec.Command(self, "watch-reply", input.SessionID)
			cmd.Start()
		}
	}
}
