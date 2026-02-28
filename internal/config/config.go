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

	// Thresholds
	SummaryThreshold    int `yaml:"summary_threshold"`
	EscalationThreshold int `yaml:"escalation_threshold"`

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
		SummaryThreshold:    2,
		EscalationThreshold: 3,
		TimeoutSeconds:      300,
		LogPath:             filepath.Join(home, "meridian-home/logs/starfix.log"),
		MemoryPath:          filepath.Join(home, "meridian-home/lex-internal/state/MEMORY.md"),
		TaskQueuePath:       filepath.Join(home, "meridian-home/lex-internal/state/TASK-QUEUE.md"),
		StatePath:           filepath.Join(home, "meridian-home/lex-internal/state/STATE.md"),
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
