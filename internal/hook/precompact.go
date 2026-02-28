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

// HandlePreCompact processes the PreCompact hook event.
func HandlePreCompact(input Input, cfg *config.Config, baseDir string) {
	s, err := state.Load(baseDir, input.SessionID)
	if err != nil {
		logEvent(cfg.LogPath, input.SessionID, "ERROR", fmt.Sprintf("load state: %v", err))
		return
	}

	s.CompactionCount++
	if err := s.WriteMarker(); err != nil {
		logEvent(cfg.LogPath, input.SessionID, "ERROR", fmt.Sprintf("write marker: %v", err))
	}
	if err := s.Save(); err != nil {
		logEvent(cfg.LogPath, input.SessionID, "ERROR", fmt.Sprintf("save state: %v", err))
	}

	logEvent(cfg.LogPath, input.SessionID, "COMPACT",
		fmt.Sprintf("compaction #%d", s.CompactionCount))

	if s.CompactionCount >= cfg.SummaryThreshold && cfg.TelegramEnabled {
		msg := fmt.Sprintf("[Starfix] Compaction #%d — session %s\nTimestamp: %s",
			s.CompactionCount, shortID(input.SessionID), time.Now().UTC().Format(time.RFC3339))
		telegram.Send(cfg.TelegramBinary, msg)
	}

	if s.CompactionCount >= cfg.EscalationThreshold && !s.EscalationPending {
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
				"[Starfix] Context pressure — session %s\nCompaction #%d this session.\nTriage: %s\nRecommended: %s\nWill %s in %ds — reply to override.",
				shortID(input.SessionID), s.CompactionCount, result.Reason, result.Action,
				result.Action, cfg.TimeoutSeconds)
			telegram.Send(cfg.TelegramBinary, msg)

			self, _ := os.Executable()
			cmd := exec.Command(self, "watch-reply", input.SessionID)
			cmd.Start()
		}
	}
}
