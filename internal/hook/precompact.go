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

// modeLabelFor returns a human-readable label for the given operational mode.
func modeLabelFor(mode operationalMode) string {
	if mode == modeRalph {
		return "ralph"
	}
	return "autonomous"
}

// lockPathFor returns the lock file path for the given operational mode.
func lockPathFor(mode operationalMode, cfg *config.Config) string {
	if mode == modeAutonomous {
		return cfg.AutonomousLockPath
	}
	return cfg.RalphLockPath
}

// detectNewLoop checks whether a new autonomous/ralph loop has started by comparing
// lock file mtime against the state file mtime. Resets the compaction counter if a
// fresh loop is detected.
func detectNewLoop(s *state.SessionState, mode operationalMode, cfg *config.Config, sessionID string) {
	lockPath := lockPathFor(mode, cfg)

	lockInfo, err := os.Stat(lockPath)
	if err != nil {
		return
	}
	stateInfo, err := os.Stat(s.StateFile())
	if err != nil {
		return
	}
	if lockInfo.ModTime().After(stateInfo.ModTime()) {
		logEvent(cfg.LogPath, sessionID, "RESET",
			fmt.Sprintf("new %s loop detected — resetting compaction count from %d",
				modeLabelFor(mode), s.CompactionCount))
		s.ResetCompactionCount()
	}
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
	s.Save()

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
		logEvent(cfg.LogPath, input.SessionID, "COMPACT", "compaction (interactive — context marker written)")
		s.Save()
		return
	}

	modeLabel := modeLabelFor(mode)

	detectNewLoop(s, mode, cfg, input.SessionID)

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
