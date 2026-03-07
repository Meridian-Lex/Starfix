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
		// Unreachable with current modes (interactive exits before thresholds() is called);
		// kept as a safety net for future mode additions.
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

// modeLabelFor returns a human-readable label for the given operational mode.
func modeLabelFor(mode operationalMode) string {
	if mode == modeRalph {
		return "ralph"
	}
	return "autonomous"
}

// detectRalphEpochReset checks whether a new ralph loop has started by comparing
// the lock file mtime against the last known epoch start. Resets the compaction
// counter and clears escalation state if a fresh loop is detected.
func detectRalphEpochReset(s *state.SessionState, cfg *config.Config, sessionID string) {
	info, err := os.Stat(cfg.RalphLockPath)
	if err != nil {
		return
	}
	if !info.ModTime().After(s.LastRalphEpochStart) {
		return
	}
	if !s.LastRalphEpochStart.IsZero() {
		// Genuine epoch transition: ralph lock was recreated since last seen.
		if s.CompactionCount > 0 {
			logEvent(cfg.LogPath, sessionID, "RESET",
				fmt.Sprintf("new ralph epoch (lock mtime %s) — resetting count from %d",
					info.ModTime().UTC().Format(time.RFC3339), s.CompactionCount))
		}
	} else if s.CompactionCount > 0 {
		// Cross-mode transition: autonomous counts accumulated, now entering ralph
		// for the first time in this session. Log so the reset is auditable.
		logEvent(cfg.LogPath, sessionID, "RESET",
			fmt.Sprintf("ralph mode entered from autonomous (count was %d)", s.CompactionCount))
	}
	s.CompactionCount = 0
	s.EscalationPending = false
	s.LastRalphEpochStart = info.ModTime()
}

// sendCompactionSummary sends a Telegram notification when the compaction count
// reaches the summary threshold.
func sendCompactionSummary(s *state.SessionState, cfg *config.Config, modeLabel, sessionID string) {
	msg := fmt.Sprintf("[Starfix] Compaction #%d (%s) — session %s\nTimestamp: %s",
		s.CompactionCount, modeLabel, shortID(sessionID),
		time.Now().UTC().Format(time.RFC3339))
	telegram.Send(cfg.TelegramBinary, msg)
}

// killExistingWatchReply terminates any previously-spawned watch-reply process
// for this session (best-effort).
func killExistingWatchReply(pidFile string) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return
	}
	if proc, err := os.FindProcess(pid); err == nil {
		proc.Kill() // best-effort; ignore error if already dead
	}
}

// spawnWatchReply starts a watch-reply subprocess and records its PID file.
func spawnWatchReply(baseDir, sessionID, logPath string) {
	pidFile := filepath.Join(baseDir, "sessions", sessionID, "watch-reply.pid")
	killExistingWatchReply(pidFile)

	self, _ := os.Executable()
	cmd := exec.Command(self, "watch-reply", sessionID)
	if err := cmd.Start(); err != nil {
		logEvent(logPath, sessionID, "ERROR", fmt.Sprintf("start watch-reply: %v", err))
		return
	}

	// Write PID file; if it fails, kill the spawned process and log the error.
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		cmd.Process.Kill()
		logEvent(logPath, sessionID, "ERROR", fmt.Sprintf("write PID file: %v", err))
	}
}

// handleEscalation performs triage assessment and notifies the fleet admiral
// when compaction count reaches the escalation threshold.
func handleEscalation(s *state.SessionState, cfg *config.Config, modeLabel, sessionID, baseDir string) {
	taskContent, _ := os.ReadFile(cfg.TaskQueuePath)
	result := triage.Assess(triage.Input{
		CompactionCount:  s.CompactionCount,
		TaskQueueContent: string(taskContent),
	})

	s.EscalationPending = true
	s.TriageDefault = result.Action
	s.EscalationSentAt = time.Now().UTC()
	if err := s.Save(); err != nil {
		logEvent(cfg.LogPath, sessionID, "ERROR", fmt.Sprintf("save escalation state: %v", err))
	}

	if !cfg.TelegramEnabled {
		return
	}

	msg := fmt.Sprintf(
		"[Starfix] Context pressure — session %s\nMode: %s | Compaction #%d\nTriage: %s\nRecommended: %s\nWill %s in %ds — reply to override.",
		shortID(sessionID), modeLabel, s.CompactionCount,
		result.Reason, result.Action, result.Action, cfg.TimeoutSeconds)
	telegram.Send(cfg.TelegramBinary, msg)

	spawnWatchReply(baseDir, sessionID, cfg.LogPath)
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

	modeLabel := modeLabelFor(mode)

	// Ralph epoch detection: if the ralph lock file has been recreated since we
	// last saw it (new loop started within the same session), reset the counter
	// and clear any prior escalation so the fresh loop gets a clean slate.
	// Note: autonomous mode intentionally has NO epoch reset — autonomous compactions
	// accumulate across the session lifetime by design. The higher autonomous thresholds
	// are calibrated for this accumulation pattern. Ralph loops are short-lived and
	// self-contained, so each new loop starts with a clean count.
	if mode == modeRalph {
		detectRalphEpochReset(s, cfg, input.SessionID)
	}

	// Atomically increment and save to prevent race conditions.
	if err := s.IncrementCompactionCount(); err != nil {
		logEvent(cfg.LogPath, input.SessionID, "ERROR", fmt.Sprintf("save state: %v", err))
	}
	logEvent(cfg.LogPath, input.SessionID, "COMPACT",
		fmt.Sprintf("compaction #%d (%s)", s.CompactionCount, modeLabel))

	summaryThreshold, escalationThreshold := thresholds(mode, cfg)

	if s.CompactionCount >= summaryThreshold && cfg.TelegramEnabled {
		sendCompactionSummary(s, cfg, modeLabel, input.SessionID)
	}

	if s.CompactionCount >= escalationThreshold && !s.EscalationPending {
		handleEscalation(s, cfg, modeLabel, input.SessionID, baseDir)
	}
}
