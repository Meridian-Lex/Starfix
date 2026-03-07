package hook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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

// lockPathFor returns the lock file path for the given operational mode.
func lockPathFor(mode operationalMode, cfg *config.Config) string {
	if mode == modeAutonomous {
		return cfg.AutonomousLockPath
	}
	return cfg.RalphLockPath
}

// detectNewLoop checks whether a new autonomous/ralph loop has started and
// resets the compaction counter if a fresh loop is detected.
//
// For ralph mode, detection delegates to detectRalphLoopReset which uses an
// inode+mtime token (LastRalphEpochToken) for stronger epoch identification.
//
// For autonomous mode, the lock file mtime is compared against the state file mtime:
// if the lock is newer than the last state write, a new autonomous loop started.
func detectNewLoop(s *state.SessionState, mode operationalMode, cfg *config.Config, sessionID string) {
	if mode == modeRalph {
		detectRalphLoopReset(s, cfg, sessionID)
		return
	}
	// Autonomous mode: compare lock mtime against state file mtime.
	lockInfo, err := os.Stat(cfg.AutonomousLockPath)
	if err != nil {
		return
	}
	stateInfo, err := os.Stat(s.StateFile())
	if err != nil {
		return
	}
	if lockInfo.ModTime().After(stateInfo.ModTime()) {
		prevCount := s.CompactionCount
		if err := s.ResetLoop(); err != nil {
			logEvent(cfg.LogPath, sessionID, "ERROR",
				fmt.Sprintf("failed to reset compaction count from %d for new autonomous loop: %v",
					prevCount, err))
		} else {
			logEvent(cfg.LogPath, sessionID, "RESET",
				fmt.Sprintf("new autonomous loop detected — reset compaction count from %d", prevCount))
		}
	}
}

// ralphEpochToken builds a stable identifier for the current RALPH-LOOP.lock
// file. It combines the inode number (from syscall.Stat_t) with the mtime in
// nanoseconds so that two lock files written within the same mtime granularity
// (e.g. on HFS+ or FAT with 1–2 s resolution) are still distinguishable.
// If the inode cannot be obtained the token falls back to mtime-only, which is
// still used for comparison — the caller must treat a change in token as a new
// epoch regardless of which fallback path was taken.
func ralphEpochToken(info os.FileInfo) string {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return fmt.Sprintf("%d:%d", st.Ino, info.ModTime().UnixNano())
	}
	// Fallback: mtime only (less robust but safe on exotic filesystems).
	return fmt.Sprintf("mtime:%d", info.ModTime().UnixNano())
}

// detectRalphLoopReset checks for a new or first-time ralph loop and resets state
// if a fresh loop is detected.
//
//   - If LastRalphEpochToken is empty (first ralph compaction this session), any
//     autonomous counts accumulated prior are cleared (cross-mode reset) and the
//     token is anchored to the current lock file.
//   - Otherwise, the current lock file's token (inode + mtime) is compared against
//     LastRalphEpochToken. A different token means the lock was recreated — a new
//     loop started. This is more robust than mtime alone because two lock files
//     created within the same mtime granularity get different inode numbers.
//   - LastRalphEpochStart is kept in sync for human-readable diagnostics and
//     JSON backward compatibility.
func detectRalphLoopReset(s *state.SessionState, cfg *config.Config, sessionID string) {
	info, err := os.Stat(cfg.RalphLockPath)
	if err != nil {
		return
	}
	token := ralphEpochToken(info)
	lockMtime := info.ModTime()

	if s.LastRalphEpochToken == "" {
		// First time entering ralph mode this session.
		if s.CompactionCount > 0 {
			// Cross-mode transition: autonomous counts accumulated before ralph started.
			logEvent(cfg.LogPath, sessionID, "RESET",
				fmt.Sprintf("ralph mode entered from autonomous (count was %d)", s.CompactionCount))
			if err := s.ResetLoop(); err != nil {
				logEvent(cfg.LogPath, sessionID, "ERROR",
					fmt.Sprintf("failed to reset on ralph entry: %v", err))
				return
			}
		}
		s.LastRalphEpochToken = token
		s.LastRalphEpochStart = lockMtime
		if err := s.Save(); err != nil {
			logEvent(cfg.LogPath, sessionID, "ERROR",
				fmt.Sprintf("failed to persist ralph epoch token: %v", err))
		}
		return
	}

	// Within an existing ralph epoch: compare token to detect lock file recreation.
	if token != s.LastRalphEpochToken {
		// Ralph lock was recreated — new loop started within this session.
		prevCount := s.CompactionCount
		if prevCount > 0 {
			logEvent(cfg.LogPath, sessionID, "RESET",
				fmt.Sprintf("new ralph epoch (token %s) — resetting count from %d",
					token, prevCount))
		}
		if err := s.ResetLoop(); err != nil {
			logEvent(cfg.LogPath, sessionID, "ERROR",
				fmt.Sprintf("failed to reset compaction count from %d for new ralph epoch: %v",
					prevCount, err))
			return
		}
		s.LastRalphEpochToken = token
		s.LastRalphEpochStart = lockMtime
		if err := s.Save(); err != nil {
			logEvent(cfg.LogPath, sessionID, "ERROR",
				fmt.Sprintf("failed to persist ralph epoch token: %v", err))
		}
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
	_ = os.Remove(pidFile) // remove stale PID file so it cannot be reused
}

// spawnWatchReply starts a watch-reply subprocess and records its PID file.
func spawnWatchReply(baseDir, sessionID, logPath string) {
	pidFile := filepath.Join(baseDir, "sessions", sessionID, "watch-reply.pid")
	killExistingWatchReply(pidFile)

	// nosemgrep: go.lang.security.audit.dangerous-exec-command
	// Safe: os.Executable() returns the path to the current binary, not user input.
	self, _ := os.Executable()
	cmd := exec.Command(self, "watch-reply", sessionID) // #nosec G204
	if err := cmd.Start(); err != nil {
		logEvent(logPath, sessionID, "ERROR", fmt.Sprintf("start watch-reply: %v", err))
		return
	}

	// Write PID file; if it fails, kill the spawned process and log the error.
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		cmd.Process.Kill()
		_ = os.Remove(pidFile)
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
		return
	}

	if !cfg.TelegramEnabled {
		return
	}

	msg := fmt.Sprintf(
		"[Starfix] Context pressure — session %s\nMode: %s | Compaction #%d\nTriage: %s\nRecommended: %s\nWill %s in %ds — reply to override.",
		shortID(sessionID), modeLabel, s.CompactionCount,
		result.Reason, result.Action, result.Action, cfg.TimeoutSeconds)
	if err := telegram.Send(cfg.TelegramBinary, msg); err != nil {
		logEvent(cfg.LogPath, sessionID, "ERROR", fmt.Sprintf("send escalation: %v", err))
	}

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
		if err := s.Save(); err != nil {
			logEvent(cfg.LogPath, input.SessionID, "ERROR", fmt.Sprintf("save session state: %v", err))
		}
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
