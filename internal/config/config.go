package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all Starfix runtime configuration.
type Config struct {
	// Context injection
	ProjectContext bool `yaml:"project_context"`

	// Telegram
	TelegramEnabled    bool   `yaml:"telegram_enabled"`
	TelegramBinary     string `yaml:"telegram_notify_binary"`
	TelegramInboundLog string `yaml:"telegram_inbound_log"`
	TelegramAdmiralID  int64  `yaml:"telegram_admiral_id"`

	// Thresholds — global fallback (used if mode-specific thresholds are zero)
	SummaryThreshold    int `yaml:"summary_threshold"`
	EscalationThreshold int `yaml:"escalation_threshold"`

	// Per-mode thresholds — only count compactions during autonomous operations.
	// Ralph loop thresholds (tighter — intra-session spin is a red flag).
	RalphSummaryThreshold    int `yaml:"ralph_summary_threshold"`
	RalphEscalationThreshold int `yaml:"ralph_escalation_threshold"`
	// Autonomous mode thresholds (more tolerant — multi-session, compaction is expected).
	AutonomousSummaryThreshold    int `yaml:"autonomous_summary_threshold"`
	AutonomousEscalationThreshold int `yaml:"autonomous_escalation_threshold"`

	// Lock file paths for mode detection
	AutonomousLockPath string `yaml:"autonomous_lock_path"`
	RalphLockPath      string `yaml:"ralph_lock_path"`

	// Timeout
	TimeoutSeconds int `yaml:"timeout_seconds"`

	// Logging
	LogPath string `yaml:"log_path"`

	// Lex paths
	MemoryPath    string `yaml:"memory_path"`
	TaskQueuePath string `yaml:"task_queue_path"`
	StatePath     string `yaml:"state_path"`
}

// defaults returns a Config with sensible default values.
func defaults() Config {
	home, _ := os.UserHomeDir()
	return Config{
		ProjectContext:      true,
		TelegramEnabled:     true,
		TelegramBinary:      filepath.Join(home, ".local/bin/telegram-notify"),
		TelegramInboundLog:  filepath.Join(home, "meridian-home/logs/telegram-inbound.log"),
		TelegramAdmiralID:   121956871,
		SummaryThreshold:              2,
		EscalationThreshold:           3,
		RalphSummaryThreshold:         4,
		RalphEscalationThreshold:      8,
		AutonomousSummaryThreshold:    6,
		AutonomousEscalationThreshold: 12,
		AutonomousLockPath:            filepath.Join(home, "meridian-home/lex-internal/state/AUTONOMOUS-MODE.lock"),
		RalphLockPath:                 filepath.Join(home, "meridian-home/lex-internal/state/RALPH-LOOP.lock"),
		TimeoutSeconds:                300,
		LogPath:                       filepath.Join(home, "meridian-home/logs/starfix.log"),
		MemoryPath:                    filepath.Join(home, "meridian-home/lex-internal/state/MEMORY.md"),
		TaskQueuePath:                 filepath.Join(home, "meridian-home/lex-internal/state/TASK-QUEUE.md"),
		StatePath:                     filepath.Join(home, "meridian-home/lex-internal/state/STATE.md"),
	}
}

// Load reads a YAML config file, applying defaults for missing fields.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	cfg := defaults()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return &cfg, nil
}

// DefaultPath returns the standard config file location.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config/starfix/starfix.cfg")
}
