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

	// Thresholds — global fallback. Used when a mode-specific threshold is zero
	// (not configured), or when code asks for a mode without explicit thresholds.
	SummaryThreshold    int `yaml:"summary_threshold"`
	EscalationThreshold int `yaml:"escalation_threshold"`

	// Per-mode thresholds — only count compactions during autonomous operations.
	// Ralph loop thresholds (tighter — intra-session spin is a red flag).
	RalphSummaryThreshold    int `yaml:"ralph_summary_threshold"`
	RalphEscalationThreshold int `yaml:"ralph_escalation_threshold"`
	// Autonomous mode thresholds (more tolerant — multi-session, compaction is expected).
	AutonomousSummaryThreshold    int `yaml:"autonomous_summary_threshold"`
	AutonomousEscalationThreshold int `yaml:"autonomous_escalation_threshold"`

	// Triage thresholds — used by triage.Assess to decide park/continue.
	// ParkAbove: unconditional park at this compaction count (default 5).
	// ContinueBelow: continue with active task at or below this count (default 2).
	// Mode-specific overrides take precedence; zero means use global/built-in default.
	TriageParkAbove     int `yaml:"triage_park_above"`
	TriageContinueBelow int `yaml:"triage_continue_below"`

	RalphTriageParkAbove     int `yaml:"ralph_triage_park_above"`
	RalphTriageContinueBelow int `yaml:"ralph_triage_continue_below"`

	AutonomousTriageParkAbove     int `yaml:"autonomous_triage_park_above"`
	AutonomousTriageContinueBelow int `yaml:"autonomous_triage_continue_below"`

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

// TriageThresholdsFor returns the park/continue thresholds for the named mode.
// Mode-specific values take precedence; falls back to global, then built-in defaults (zero).
func (c *Config) TriageThresholdsFor(mode string) (parkAbove, continueBelow int) {
	switch mode {
	case "ralph":
		parkAbove, continueBelow = c.RalphTriageParkAbove, c.RalphTriageContinueBelow
	case "autonomous":
		parkAbove, continueBelow = c.AutonomousTriageParkAbove, c.AutonomousTriageContinueBelow
	}
	if parkAbove <= 0 {
		parkAbove = c.TriageParkAbove
	}
	if continueBelow <= 0 {
		continueBelow = c.TriageContinueBelow
	}
	return
}

// DefaultPath returns the standard config file location.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config/starfix/starfix.cfg")
}
