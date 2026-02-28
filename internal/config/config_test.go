package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/meridian-lex/starfix/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "starfix.cfg")
	os.WriteFile(cfgPath, []byte("telegram_enabled: false\n"), 0644)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.SummaryThreshold != 2 {
		t.Errorf("SummaryThreshold: got %d, want 2", cfg.SummaryThreshold)
	}
	if cfg.EscalationThreshold != 3 {
		t.Errorf("EscalationThreshold: got %d, want 3", cfg.EscalationThreshold)
	}
	if cfg.TimeoutSeconds != 300 {
		t.Errorf("TimeoutSeconds: got %d, want 300", cfg.TimeoutSeconds)
	}
	if cfg.ProjectContext != true {
		t.Errorf("ProjectContext: got false, want true")
	}
}

func TestLoad_Override(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "starfix.cfg")
	os.WriteFile(cfgPath, []byte("summary_threshold: 5\nescalation_threshold: 7\n"), 0644)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.SummaryThreshold != 5 {
		t.Errorf("got %d, want 5", cfg.SummaryThreshold)
	}
	if cfg.EscalationThreshold != 7 {
		t.Errorf("got %d, want 7", cfg.EscalationThreshold)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load("/nonexistent/path/starfix.cfg")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestDefaultPath(t *testing.T) {
	path := config.DefaultPath()
	if path == "" {
		t.Fatal("DefaultPath returned empty string")
	}
}
