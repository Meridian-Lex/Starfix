package hook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
// Mode-specific values take precedence; if a mode-specific threshold is zero
// (unset in config), the global fallback is used instead.
func thresholds(mode operationalMode, cfg *config.Config) (summary, escalation int) {
	switch mode {
	case modeRalph:
		summary, escalation = cfg.RalphSummaryThreshold, cfg.RalphEscalationThreshold
	case modeAutonomous:
		summary, escalation = cfg.AutonomousSummaryThreshold, cfg.AutonomousEscalationThreshold
	default:
		// General fallback for any mode without explicit thresholds (including interactive).
		return cfg.SummaryThreshold, cfg.EscalationThreshold
	}

	// If a mode-specific threshold is zero (not configured), fall back to the global value.
	if summary == 0 {
		summary = cfg.SummaryThreshold
	}
	if escalation == 0 {
		escalation = cfg.EscalationThreshold
	}
	return
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	// Permission errors or other stat errors: treat as present but unreadable.
	// Use stderr since logEvent requires a valid logPath which is not available here.
	fmt.Fprintf(os.Stderr, "[starfix] WARN: stat %s: %v (treating as present)\n", path, err)
	return true
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
		return
	}

	// Ralph epoch detection: if the ralph lock file has been recreated since we
	// last saw it (new loop started within the same session), reset the counter
	// and clear any prior escalation so the fresh loop gets a clean slate.
	if mode == modeRalph {
		if info, err := os.Stat(cfg.RalphLockPath); err == nil {
			if info.ModTime().After(s.LastRalphEpochStart) {
				// Only log a RESET when this is a genuine epoch transition
				// (not the first-ever ralph compaction where LastRalphEpochStart is zero).
				if s.CompactionCount > 0 && !s.LastRalphEpochStart.IsZero() {
					logEvent(cfg.LogPath, input.SessionID, "RESET",
						fmt.Sprintf("new ralph epoch (lock mtime %s) — resetting count from %d",
							info.ModTime().UTC().Format(time.RFC3339), s.CompactionCount))
				}
				s.CompactionCount = 0
				s.EscalationPending = false
				s.LastRalphEpochStart = info.ModTime()
			}
		}
	}

	// Atomically increment and save to prevent race conditions.
	if err := s.IncrementCompactionCount(); err != nil {
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

			// Kill any existing watch-reply for this session before spawning a new one.
			pidFile := filepath.Join(baseDir, "sessions", input.SessionID, "watch-reply.pid")
			if data, err := os.ReadFile(pidFile); err == nil {
				if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
					if proc, err := os.FindProcess(pid); err == nil {
						proc.Kill() // best-effort; ignore error if already dead
					}
				}
			}

			self, _ := os.Executable()
			cmd := exec.Command(self, "watch-reply", input.SessionID)
			if err := cmd.Start(); err != nil {
				logEvent(cfg.LogPath, input.SessionID, "ERROR", fmt.Sprintf("start watch-reply: %v", err))
				return
			}

			// Write PID file; if it fails, kill the spawned process and log the error.
			if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
				cmd.Process.Kill()
				logEvent(cfg.LogPath, input.SessionID, "ERROR", fmt.Sprintf("write PID file: %v", err))
			}
		}
	}
}
