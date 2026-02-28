# Starfix Implementation Plan

> **For Lex:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI binary and bash installer that restores Lex orientation after context compaction, tracks compaction frequency, and escalates to Fleet Admiral via Telegram when sessions are under pressure.

<!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
**Architecture:** Three Claude Code hooks (PreCompact, SessionStart, UserPromptSubmit) delegate to a `starfix` Go binary. All per-session state lives in `~/.config/starfix/sessions/<session_id>/`. A background `starfix watch-reply` process handles Telegram reply polling and timeout execution asynchronously. A menu-driven `install.sh` builds the binary, installs hook scripts, and registers hooks in `~/.claude/settings.json`.

**Tech Stack:** Go 1.24 (cobra for CLI), bash hooks, YAML config (gopkg.in/yaml.v3), standard library only for everything else.

---

## Parallelization Map

These task groups can be executed in parallel by separate subagents:

```
Group 1 (no deps):     Task 1 (scaffold)
Group 2 (after T1):    Task 2 (config), Task 3 (state) — parallel
Group 3 (after T2,T3): Task 4 (context/core), Task 5 (context/project),
                        Task 6 (triage), Task 7 (telegram) — parallel
Group 4 (after T4-T7): Task 8 (hook/precompact), Task 9 (hook/sessionstart),
                        Task 10 (hook/userpromptsubmit), Task 11 (watch-reply) — parallel
Group 5 (after T8-T11): Task 12 (bash hook scripts)
Group 6 (after T12):   Task 13 (install.sh), Task 14 (config file) — parallel
Group 7 (after all):   Task 15 (README + integration test)
```

---

## Context: Key Paths

```
Repo root:            ~/meridian-home/projects/Starfix/
Design doc:           docs/plans/2026-02-28-starfix-design.md
Go module:            github.com/meridian-lex/starfix
Runtime config:       ~/.config/starfix/starfix.cfg
Runtime sessions:     ~/.config/starfix/sessions/<session_id>/
Fleet log:            ~/meridian-home/logs/starfix.log
Telegram binary:      ~/.local/bin/telegram-notify
Telegram inbound log: ~/meridian-home/logs/telegram-inbound.log
Existing hooks dir:   ~/meridian-home/lex-internal/enforcement/hooks/
<!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
Settings file:        ~/.claude/settings.json
```

## Context: Hook Stdin Format

Every hook receives JSON on stdin with at minimum:
```json
{
  "session_id": "abc123def456...",
  "hook_event_name": "PreCompact",
  "cwd": "/current/working/dir"
}
```

## Context: SessionStart Output Format

SessionStart hook must output this exact JSON to inject context:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
"additionalContext": "text to inject into Lex context"
  }
}
```

## Context: Telegram Inbound Log Format

Each line in `~/meridian-home/logs/telegram-inbound.log` is a JSON object:
```json
{"timestamp": "2026-02-28T14:32:00Z", "from": {"id": 121956871}, "text": "continue"}
```
Parse lines newer than the escalation timestamp and match on `from.id == 121956871` (Fleet Admiral's Telegram ID — read from config).

---

## Task 1: Project Scaffold

**Files:**
- Create: `go.mod`
- Create: `go.sum` (generated)
- Create: `cmd/starfix/main.go`
- Create: `Makefile`
- Create: `internal/hook/.gitkeep`
- Create: `internal/context/.gitkeep`
- Create: `internal/triage/.gitkeep`
- Create: `internal/telegram/.gitkeep`
- Create: `internal/state/.gitkeep`
- Create: `internal/config/.gitkeep`
- Create: `hooks/.gitkeep`
- Create: `config/.gitkeep`

**Step 1: Initialize Go module**

```bash
cd ~/meridian-home/projects/Starfix
go mod init github.com/meridian-lex/starfix
```

Expected: `go.mod` created with `module github.com/meridian-lex/starfix` and `go 1.24`

**Step 2: Add cobra dependency**

```bash
go get github.com/spf13/cobra@latest
go get gopkg.in/yaml.v3@latest
go mod tidy
```

**Step 3: Create directory structure**

```bash
mkdir -p cmd/starfix internal/hook internal/context internal/triage \
         internal/telegram internal/state internal/config hooks config
```

**Step 4: Write `cmd/starfix/main.go`**

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "starfix",
		Short: "Post-compaction context restoration for Lex",
	}

	hookCmd := &cobra.Command{
		Use:   "hook",
		Short: "Hook subcommands",
	}

	hookCmd.AddCommand(
		&cobra.Command{
			Use:   "precompact",
			Short: "Handle PreCompact hook event",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Fprintln(os.Stderr, "precompact: not implemented")
				return nil
			},
		},
		&cobra.Command{
			Use:   "sessionstart",
			Short: "Handle SessionStart hook event",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Fprintln(os.Stderr, "sessionstart: not implemented")
				return nil
			},
		},
		&cobra.Command{
			Use:   "userpromptsubmit",
			Short: "Handle UserPromptSubmit hook event",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Fprintln(os.Stderr, "userpromptsubmit: not implemented")
				return nil
			},
		},
	)

	watchCmd := &cobra.Command{
		Use:   "watch-reply [session_id]",
		Short: "Watch for Telegram reply and execute timeout action",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stderr, "watch-reply: not implemented")
			return nil
		},
	}

	root.AddCommand(hookCmd, watchCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
```

**Step 5: Write `Makefile`**

```makefile
BINARY := starfix
BUILD_DIR := bin
CMD := ./cmd/starfix

.PHONY: build test lint vet clean install

build:
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD)

test:
	go test ./...

lint:
	go vet ./...

vet:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)

install: build
	cp $(BUILD_DIR)/$(BINARY) ~/.local/bin/$(BINARY)
```

**Step 6: Verify build**

```bash
cd ~/meridian-home/projects/Starfix
make build
./bin/starfix --help
```

Expected: help output with hook and watch-reply subcommands listed.

**Step 7: Commit**

```bash
git add go.mod go.sum cmd/ internal/ hooks/ config/ Makefile
git commit -m "feat: scaffold Starfix project structure"
```

---

## Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write failing test**

```go
// internal/config/config_test.go
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
	// Write minimal config
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
```

**Step 2: Run to verify fail**

```bash
cd ~/meridian-home/projects/Starfix
go test ./internal/config/... 2>&1
```

Expected: compile error — package does not exist.

**Step 3: Implement `internal/config/config.go`**

```go
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
	TelegramEnabled      bool   `yaml:"telegram_enabled"`
	TelegramBinary       string `yaml:"telegram_notify_binary"`
	TelegramInboundLog   string `yaml:"telegram_inbound_log"`
	TelegramAdmiralID    int64  `yaml:"telegram_admiral_id"`

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
```

**Step 4: Run tests to verify pass**

```bash
go test ./internal/config/... -v
```

Expected: all tests PASS.

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package with YAML loading and defaults"
```

---

## Task 3: State Package

**Files:**
- Create: `internal/state/state.go`
- Create: `internal/state/state_test.go`

**Step 1: Write failing tests**

```go
// internal/state/state_test.go
package state_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/meridian-lex/starfix/internal/state"
)

func TestState_LoadEmpty(t *testing.T) {
	dir := t.TempDir()
	s, err := state.Load(dir, "session-abc")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s.CompactionCount != 0 {
		t.Errorf("CompactionCount: got %d, want 0", s.CompactionCount)
	}
	if s.EscalationPending {
		t.Error("EscalationPending: got true, want false")
	}
}

func TestState_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	s, _ := state.Load(dir, "session-abc")
	s.CompactionCount = 3
	s.EscalationPending = true
	s.TriageDefault = "park"
	s.EscalationSentAt = time.Now().UTC()

	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	s2, err := state.Load(dir, "session-abc")
	if err != nil {
		t.Fatalf("Load after save failed: %v", err)
	}
	if s2.CompactionCount != 3 {
		t.Errorf("CompactionCount: got %d, want 3", s2.CompactionCount)
	}
	if !s2.EscalationPending {
		t.Error("EscalationPending: got false, want true")
	}
	if s2.TriageDefault != "park" {
		t.Errorf("TriageDefault: got %q, want park", s2.TriageDefault)
	}
}

func TestMarker_WriteAndCheck(t *testing.T) {
	dir := t.TempDir()
	s, _ := state.Load(dir, "session-xyz")

	if s.MarkerExists() {
		t.Error("marker should not exist before write")
	}

	if err := s.WriteMarker(); err != nil {
		t.Fatalf("WriteMarker failed: %v", err)
	}
	if !s.MarkerExists() {
		t.Error("marker should exist after write")
	}

	s.DeleteMarker()
	if s.MarkerExists() {
		t.Error("marker should not exist after delete")
	}
}

func TestState_SessionDir(t *testing.T) {
	dir := t.TempDir()
	s, _ := state.Load(dir, "session-abc")
	expected := filepath.Join(dir, "sessions", "session-abc")
	if s.Dir() != expected {
		t.Errorf("Dir: got %q, want %q", s.Dir(), expected)
	}
}
```

**Step 2: Run to verify fail**

```bash
go test ./internal/state/... 2>&1
```

Expected: compile error.

**Step 3: Implement `internal/state/state.go`**

```go
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// SessionState holds per-session compaction and escalation state.
type SessionState struct {
	CompactionCount   int       `json:"compaction_count"`
	EscalationPending bool      `json:"escalation_pending"`
	TriageDefault     string    `json:"triage_default"` // "continue" or "park"
	EscalationSentAt  time.Time `json:"escalation_sent_at,omitempty"`
	ReplyReceived     bool      `json:"reply_received"`
	ReplyText         string    `json:"reply_text"`
	TimeoutFired      bool      `json:"timeout_fired"`
	TimeoutAction     string    `json:"timeout_action"` // "continue" or "park"

	baseDir   string
	sessionID string
}

// Load reads session state from disk, returning defaults if no file exists.
func Load(baseDir, sessionID string) (*SessionState, error) {
	s := &SessionState{baseDir: baseDir, sessionID: sessionID}
	if err := os.MkdirAll(s.Dir(), 0755); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(s.stateFile())
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	s.baseDir = baseDir
	s.sessionID = sessionID
	return s, nil
}

// Save writes session state to disk.
func (s *SessionState) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.stateFile(), data, 0644)
}

// Dir returns the session-specific directory.
func (s *SessionState) Dir() string {
	return filepath.Join(s.baseDir, "sessions", s.sessionID)
}

func (s *SessionState) stateFile() string {
	return filepath.Join(s.Dir(), "state.json")
}

func (s *SessionState) markerFile() string {
	return filepath.Join(s.Dir(), "compact-pending")
}

// WriteMarker creates the compact-pending marker file.
func (s *SessionState) WriteMarker() error {
	return os.WriteFile(s.markerFile(), []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
}

// MarkerExists reports whether the compact-pending marker is present.
func (s *SessionState) MarkerExists() bool {
	_, err := os.Stat(s.markerFile())
	return err == nil
}

// DeleteMarker removes the compact-pending marker file.
func (s *SessionState) DeleteMarker() {
	os.Remove(s.markerFile())
}

// DefaultBaseDir returns ~/.config/starfix
func DefaultBaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "starfix")
}
```

**Step 4: Run tests to verify pass**

```bash
go test ./internal/state/... -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/state/
git commit -m "feat: add state package for per-session compaction tracking"
```

---

## Task 4: Context Assembly — Core

**Files:**
- Create: `internal/context/core.go`
- Create: `internal/context/core_test.go`

**Step 1: Write failing tests**

```go
// internal/context/core_test.go
package context_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	starfixctx "github.com/meridian-lex/starfix/internal/context"
	"github.com/meridian-lex/starfix/internal/config"
)

func TestBuildCore_AllPresent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "MEMORY.md"), "# Memory\nsome memory content")
	writeFile(t, filepath.Join(dir, "TASK-QUEUE.md"), "# Tasks\n- task 1")
	writeFile(t, filepath.Join(dir, "STATE.md"), "# State\nactive")

	cfg := &config.Config{
		MemoryPath:    filepath.Join(dir, "MEMORY.md"),
		TaskQueuePath: filepath.Join(dir, "TASK-QUEUE.md"),
		StatePath:     filepath.Join(dir, "STATE.md"),
	}

	payload := starfixctx.BuildCore(cfg)

	if !strings.Contains(payload, "MEMORY.md") {
		t.Error("payload should contain MEMORY.md header")
	}
	if !strings.Contains(payload, "some memory content") {
		t.Error("payload should contain memory content")
	}
	if !strings.Contains(payload, "task 1") {
		t.Error("payload should contain task queue content")
	}
	if !strings.Contains(payload, "STATE.md") {
		t.Error("payload should contain STATE.md header")
	}
}

func TestBuildCore_MissingFilesSkipped(t *testing.T) {
	cfg := &config.Config{
		MemoryPath:    "/nonexistent/MEMORY.md",
		TaskQueuePath: "/nonexistent/TASK-QUEUE.md",
		StatePath:     "/nonexistent/STATE.md",
	}
	// Should not panic; returns empty-ish payload
	payload := starfixctx.BuildCore(cfg)
	_ = payload // no crash = pass
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	os.WriteFile(path, []byte(content), 0644)
}
```

**Step 2: Run to verify fail**

```bash
go test ./internal/context/... 2>&1
```

Expected: compile error.

**Step 3: Implement `internal/context/core.go`**

```go
package context

import (
	"fmt"
	"os"
	"strings"

	"github.com/meridian-lex/starfix/internal/config"
)

// BuildCore assembles the core context payload (MEMORY.md, TASK-QUEUE.md, STATE.md).
// Missing files are silently skipped.
func BuildCore(cfg *config.Config) string {
	var b strings.Builder
	b.WriteString("=== STARFIX: POST-COMPACTION CONTEXT RESTORATION ===\n\n")
	b.WriteString("Compaction occurred. Re-orienting from fleet records.\n\n")

	appendFile(&b, "MEMORY.md", cfg.MemoryPath)
	appendFile(&b, "TASK-QUEUE.md", cfg.TaskQueuePath)
	appendFile(&b, "STATE.md", cfg.StatePath)

	b.WriteString("=== END STARFIX CONTEXT ===\n")
	return b.String()
}

func appendFile(b *strings.Builder, label, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	fmt.Fprintf(b, "--- %s ---\n%s\n\n", label, strings.TrimSpace(string(data)))
}
```

**Step 4: Run tests to verify pass**

```bash
go test ./internal/context/... -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/context/
git commit -m "feat: add context/core — assemble MEMORY/TASK-QUEUE/STATE payload"
```

---

## Task 5: Context Assembly — Project Layer

**Files:**
- Modify: `internal/context/core.go` (add `BuildProject`)
- Create: `internal/context/project.go`
- Modify: `internal/context/core_test.go` (add project tests)

**Step 1: Write failing tests**

Add to `internal/context/core_test.go`:

```go
func TestBuildProject_WithGit(t *testing.T) {
	dir := t.TempDir()

	// Init a git repo in dir
	exec.Command("git", "-C", dir, "init").Run()
	exec.Command("git", "-C", dir, "commit", "--allow-empty", "-m", "init").Run()
 <!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), "# Project rules\ndo not break things")

	payload := starfixctx.BuildProject(dir)

	if !strings.Contains(payload, "Project rules") {
  <!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
		t.Error("payload should contain CLAUDE.md content")
	}
}

func TestBuildProject_NoProjectDir(t *testing.T) {
	payload := starfixctx.BuildProject("")
	if payload != "" {
		t.Errorf("expected empty payload for empty dir, got: %q", payload)
	}
}

func TestBuildProject_NoClaudeMd(t *testing.T) {
	dir := t.TempDir()
 <!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
	// No CLAUDE.md, no README — should return empty without panic
	payload := starfixctx.BuildProject(dir)
	_ = payload
}
```

Add import: `"os/exec"` to test file.

**Step 2: Run to verify fail**

```bash
go test ./internal/context/... 2>&1
```

Expected: compile error on `BuildProject`.

**Step 3: Implement `internal/context/project.go`**

```go
package context

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BuildProject assembles the optional project-layer context payload.
// Returns empty string if projectDir is empty or no relevant files are found.
func BuildProject(projectDir string) string {
	if projectDir == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString("--- PROJECT CONTEXT ---\n")

	added := 0
 <!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
	added += appendProjectFile(&b, "CLAUDE.md", filepath.Join(projectDir, "CLAUDE.md"))
	added += appendProjectFile(&b, "README.md", filepath.Join(projectDir, "README.md"))

	if log := recentGitLog(projectDir); log != "" {
		fmt.Fprintf(&b, "--- recent git log ---\n%s\n\n", log)
		added++
	}

	if added == 0 {
		return ""
	}
	return b.String()
}

func appendProjectFile(b *strings.Builder, label, path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 50 {
		lines = lines[:50]
	}
	fmt.Fprintf(b, "--- %s (first 50 lines) ---\n%s\n\n", label, strings.Join(lines, "\n"))
	return 1
}

func recentGitLog(dir string) string {
	out, err := exec.Command("git", "-C", dir, "log", "--oneline", "-10").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
```

**Step 4: Run tests to verify pass**

```bash
go test ./internal/context/... -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/context/
<!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
git commit -m "feat: add context/project — CLAUDE.md, README, git log layer"
```

---

## Task 6: Triage Package

**Files:**
- Create: `internal/triage/triage.go`
- Create: `internal/triage/triage_test.go`

**Step 1: Write failing tests**

```go
// internal/triage/triage_test.go
package triage_test

import (
	"testing"

	"github.com/meridian-lex/starfix/internal/triage"
)

func TestAssess_HighCount_NoTask(t *testing.T) {
	result := triage.Assess(triage.Input{
		CompactionCount: 6,
		TaskQueueContent: "",
	})
	if result.Action != "park" {
		t.Errorf("Action: got %q, want park for high count + no task", result.Action)
	}
	if result.Reason == "" {
		t.Error("Reason should not be empty")
	}
}

func TestAssess_LowCount_ActiveTask(t *testing.T) {
	result := triage.Assess(triage.Input{
		CompactionCount: 2,
		TaskQueueContent: "- [in_progress] Implement feature X\n  Clear completion: yes",
	})
	if result.Action != "continue" {
		t.Errorf("Action: got %q, want continue for low count + active task", result.Action)
	}
}

func TestAssess_HighCount_ActiveTask(t *testing.T) {
	result := triage.Assess(triage.Input{
		CompactionCount: 5,
		TaskQueueContent: "- [in_progress] Implement feature X",
	})
	// High count with active task → park
	if result.Action != "park" {
		t.Errorf("Action: got %q, want park for count>=5", result.Action)
	}
}

func TestAssess_AlwaysHasReason(t *testing.T) {
	cases := []triage.Input{
		{CompactionCount: 0},
		{CompactionCount: 3},
		{CompactionCount: 10, TaskQueueContent: "something"},
	}
	for _, c := range cases {
		r := triage.Assess(c)
		if r.Reason == "" {
			t.Errorf("Reason empty for input %+v", c)
		}
	}
}
```

**Step 2: Run to verify fail**

```bash
go test ./internal/triage/... 2>&1
```

Expected: compile error.

**Step 3: Implement `internal/triage/triage.go`**

```go
package triage

import (
	"fmt"
	"strings"
)

// Input holds the signals used to assess the session situation.
type Input struct {
	CompactionCount  int
	TaskQueueContent string
}

// Result holds the triage recommendation.
type Result struct {
	Action string // "continue" or "park"
	Reason string
}

// Assess determines whether Lex should continue or park based on session signals.
func Assess(in Input) Result {
	// High compaction count is the strongest signal regardless of task state
	if in.CompactionCount >= 5 {
		return Result{
			Action: "park",
			Reason: fmt.Sprintf("compaction count is %d — session under heavy context pressure", in.CompactionCount),
		}
	}

	hasActiveTask := strings.Contains(in.TaskQueueContent, "in_progress")

	if in.CompactionCount <= 2 && hasActiveTask {
		return Result{
			Action: "continue",
			Reason: fmt.Sprintf("compaction count is %d with an active task in progress — proceeding", in.CompactionCount),
		}
	}

	if !hasActiveTask {
		return Result{
			Action: "continue",
			Reason: "no active task in progress — safe to continue",
		}
	}

	// Mid-range count with active task: park to be safe
	return Result{
		Action: "park",
		Reason: fmt.Sprintf("compaction count is %d with active work — recommend parking to avoid context drift", in.CompactionCount),
	}
}
```

**Step 4: Run tests to verify pass**

```bash
go test ./internal/triage/... -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/triage/
git commit -m "feat: add triage package — continue/park decision logic"
```

---

## Task 7: Telegram Package

**Files:**
- Create: `internal/telegram/telegram.go`
- Create: `internal/telegram/telegram_test.go`

**Step 1: Write failing tests**

```go
// internal/telegram/telegram_test.go
package telegram_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/meridian-lex/starfix/internal/telegram"
)

func TestParseInboundLog_FindsReply(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "telegram-inbound.log")

	since := time.Now().Add(-time.Hour)

	entries := []map[string]interface{}{
		{"timestamp": time.Now().UTC().Format(time.RFC3339), "from": map[string]interface{}{"id": float64(121956871)}, "text": "continue"},
	}
	var lines []byte
	for _, e := range entries {
		b, _ := json.Marshal(e)
		lines = append(lines, b...)
		lines = append(lines, '\n')
	}
	os.WriteFile(logPath, lines, 0644)

	reply, found := telegram.CheckInbound(logPath, since, 121956871)
	if !found {
		t.Fatal("expected reply to be found")
	}
	if reply != "continue" {
		t.Errorf("reply: got %q, want continue", reply)
	}
}

func TestParseInboundLog_BeforeSince_NotFound(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "telegram-inbound.log")

	since := time.Now().Add(time.Hour) // future — nothing before this

	entry := map[string]interface{}{
		"timestamp": time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
		"from": map[string]interface{}{"id": float64(121956871)},
		"text": "park",
	}
	b, _ := json.Marshal(entry)
	os.WriteFile(logPath, append(b, '\n'), 0644)

	_, found := telegram.CheckInbound(logPath, since, 121956871)
	if found {
		t.Fatal("should not find reply before since timestamp")
	}
}

func TestParseInboundLog_WrongSender(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "telegram-inbound.log")

	since := time.Now().Add(-time.Hour)
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"from": map[string]interface{}{"id": float64(999999)}, // wrong ID
		"text": "continue",
	}
	b, _ := json.Marshal(entry)
	os.WriteFile(logPath, append(b, '\n'), 0644)

	_, found := telegram.CheckInbound(logPath, since, 121956871)
	if found {
		t.Fatal("should not find reply from wrong sender")
	}
}

func TestParseInboundLog_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "empty.log")
	os.WriteFile(logPath, []byte{}, 0644)

	_, found := telegram.CheckInbound(logPath, time.Now().Add(-time.Hour), 121956871)
	if found {
		t.Fatal("should not find reply in empty file")
	}
}
```

**Step 2: Run to verify fail**

```bash
go test ./internal/telegram/... 2>&1
```

**Step 3: Implement `internal/telegram/telegram.go`**

```go
package telegram

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"time"
)

// Send sends a Telegram message using the fleet notify binary.
func Send(binary, message string) error {
	return exec.Command(binary, message).Run()
}

// inboundEntry represents one line in the Telegram inbound log.
type inboundEntry struct {
	Timestamp string `json:"timestamp"`
	From      struct {
		ID int64 `json:"id"`
	} `json:"from"`
	Text string `json:"text"`
}

// CheckInbound scans the inbound log for a reply from admiralID after since.
// Returns the reply text and true if found.
func CheckInbound(logPath string, since time.Time, admiralID int64) (string, bool) {
	f, err := os.Open(logPath)
	if err != nil {
		return "", false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry inboundEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		ts, err := time.Parse(time.RFC3339, entry.Timestamp)
		if err != nil {
			continue
		}
		if ts.After(since) && entry.From.ID == admiralID {
			return entry.Text, true
		}
	}
	return "", false
}
```

**Step 4: Run tests to verify pass**

```bash
go test ./internal/telegram/... -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/telegram/
git commit -m "feat: add telegram package — send and inbound log parsing"
```

---

## Task 8: Hook Handler — PreCompact

**Files:**
- Create: `internal/hook/precompact.go`
- Create: `internal/hook/precompact_test.go`
- Modify: `cmd/starfix/main.go`

**Step 1: Write failing tests**

```go
// internal/hook/precompact_test.go
package hook_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meridian-lex/starfix/internal/config"
	"github.com/meridian-lex/starfix/internal/hook"
	"github.com/meridian-lex/starfix/internal/state"
)

func TestPreCompact_WritesMarker(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-test-1")

	hook.HandlePreCompact(input, cfg, dir)

	s, _ := state.Load(dir, "session-test-1")
	if !s.MarkerExists() {
		t.Error("marker file should exist after precompact")
	}
}

func TestPreCompact_IncrementsCount(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-test-2")

	hook.HandlePreCompact(input, cfg, dir)
	hook.HandlePreCompact(input, cfg, dir)

	s, _ := state.Load(dir, "session-test-2")
	if s.CompactionCount != 2 {
		t.Errorf("CompactionCount: got %d, want 2", s.CompactionCount)
	}
}

func TestPreCompact_SetsEscalationAtThreshold(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.EscalationThreshold = 2
	cfg.TelegramEnabled = false // don't actually send
	input := hookInput("session-test-3")

	hook.HandlePreCompact(input, cfg, dir)
	hook.HandlePreCompact(input, cfg, dir)

	s, _ := state.Load(dir, "session-test-3")
	if !s.EscalationPending {
		t.Error("EscalationPending should be true at threshold")
	}
	if s.TriageDefault != "continue" && s.TriageDefault != "park" {
		t.Errorf("TriageDefault should be set, got %q", s.TriageDefault)
	}
}

func hookInput(sessionID string) hook.Input {
	return hook.Input{SessionID: sessionID, CWD: "/tmp"}
}

func testConfig(dir string) *config.Config {
	return &config.Config{
		SummaryThreshold:    2,
		EscalationThreshold: 3,
		TimeoutSeconds:      5,
		TelegramEnabled:     false,
		LogPath:             filepath.Join(dir, "starfix.log"),
		TaskQueuePath:       filepath.Join(dir, "TASK-QUEUE.md"),
	}
}
```

**Step 2: Run to verify fail**

```bash
go test ./internal/hook/... 2>&1
```

**Step 3: Implement `internal/hook/precompact.go`**

```go
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

	if !cfg.TelegramEnabled {
		return
	}

	if s.CompactionCount >= cfg.SummaryThreshold {
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

		msg := fmt.Sprintf(
			"[Starfix] Context pressure — session %s\nCompaction #%d this session.\nTriage: %s\nRecommended: %s\nWill %s in %ds — reply to override.",
			shortID(input.SessionID), s.CompactionCount, result.Reason, result.Action,
			result.Action, cfg.TimeoutSeconds)
		telegram.Send(cfg.TelegramBinary, msg)

		// Spawn background reply watcher
		self, _ := os.Executable()
		cmd := exec.Command(self, "watch-reply", input.SessionID)
		cmd.Start()
	}
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
```

**Step 4: Create `internal/hook/types.go`** (shared input type)

```go
package hook

import "encoding/json"

// Input holds the hook event data from stdin.
type Input struct {
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`
}

// ReadInput parses hook stdin JSON into Input.
func ReadInput(data []byte) (Input, error) {
	var in Input
	return in, json.Unmarshal(data, &in)
}
```

**Step 5: Create `internal/hook/log.go`**

```go
package hook

import (
	"fmt"
	"os"
	"time"
)

func logEvent(logPath, sessionID, event, message string) {
	line := fmt.Sprintf("%s [%s] %s %s\n",
		time.Now().UTC().Format(time.RFC3339),
		shortID(sessionID), event, message)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(line)
}
```

**Step 6: Run tests to verify pass**

```bash
go test ./internal/hook/... -v
```

Expected: all PASS.

**Step 7: Commit**

```bash
git add internal/hook/
git commit -m "feat: add hook/precompact handler — marker, counter, Telegram escalation"
```

---

## Task 9: Hook Handler — SessionStart

**Files:**
- Create: `internal/hook/sessionstart.go`
- Modify: `internal/hook/precompact_test.go` (add sessionstart tests)

**Step 1: Write failing tests**

Create `internal/hook/sessionstart_test.go`:

```go
package hook_test

import (
	"strings"
	"testing"

	"github.com/meridian-lex/starfix/internal/hook"
	"github.com/meridian-lex/starfix/internal/state"
)

func TestSessionStart_NoMarker_EmptyOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)

	output := hook.HandleSessionStart(hookInput("session-ss-1"), cfg, dir)
	if output != "" {
		t.Errorf("expected empty output when no marker, got: %q", output)
	}
}

func TestSessionStart_WithMarker_InjectsContext(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ss-2")

	// Write a marker to simulate post-compaction state
	s, _ := state.Load(dir, "session-ss-2")
	s.WriteMarker()

	output := hook.HandleSessionStart(input, cfg, dir)

	if !strings.Contains(output, "STARFIX") {
		t.Errorf("output should contain STARFIX header, got: %q", output)
	}
}

func TestSessionStart_WithMarker_DeletesMarker(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ss-3")

	s, _ := state.Load(dir, "session-ss-3")
	s.WriteMarker()

	hook.HandleSessionStart(input, cfg, dir)

	s2, _ := state.Load(dir, "session-ss-3")
	if s2.MarkerExists() {
		t.Error("marker should be deleted after sessionstart")
	}
}

func TestSessionStart_OutputIsValidJSON(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ss-4")

	s, _ := state.Load(dir, "session-ss-4")
	s.WriteMarker()

	output := hook.HandleSessionStart(input, cfg, dir)
	if output == "" {
		t.Fatal("expected non-empty JSON output")
	}
	// Must be valid JSON
	var v interface{}
	if err := json.Unmarshal([]byte(output), &v); err != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", err, output)
	}
}
```

Add `"encoding/json"` import to test file.

**Step 2: Run to verify fail**

```bash
go test ./internal/hook/... 2>&1
```

**Step 3: Implement `internal/hook/sessionstart.go`**

```go
package hook

import (
	"encoding/json"
	"fmt"
	"os"

	starfixctx "github.com/meridian-lex/starfix/internal/context"
	"github.com/meridian-lex/starfix/internal/config"
	"github.com/meridian-lex/starfix/internal/state"
)

type sessionStartOutput struct {
	HookSpecificOutput sessionStartSpecific `json:"hookSpecificOutput"`
}

type sessionStartSpecific struct {
	HookEventName    string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// HandleSessionStart processes the SessionStart hook event.
// Returns JSON string for additionalContext injection, or empty string if no action needed.
func HandleSessionStart(input Input, cfg *config.Config, baseDir string) string {
	s, err := state.Load(baseDir, input.SessionID)
	if err != nil {
		logEvent(cfg.LogPath, input.SessionID, "ERROR", fmt.Sprintf("sessionstart load state: %v", err))
		return ""
	}

	if !s.MarkerExists() {
		return ""
	}

	// Build context payload
	payload := starfixctx.BuildCore(cfg)
	if cfg.ProjectContext && input.CWD != "" {
		if projectLayer := starfixctx.BuildProject(input.CWD); projectLayer != "" {
			payload += "\n" + projectLayer
		}
	}

	// Check for pending reply/timeout from watch-reply process
	if s.ReplyReceived {
		payload += fmt.Sprintf("\n--- ADMIRAL REPLY ---\nFleet Admiral replied: %s\n", s.ReplyText)
		s.ReplyReceived = false
		s.ReplyText = ""
		s.EscalationPending = false
	} else if s.TimeoutFired {
		if s.TimeoutAction == "park" {
			payload += "\n--- STARFIX DIRECTIVE ---\nNo Admiral reply received within timeout. Triage recommended PARK. Please wrap up current work and stop.\n"
		}
		s.TimeoutFired = false
		s.EscalationPending = false
	}

	s.DeleteMarker()
	s.Save()

	logEvent(cfg.LogPath, input.SessionID, "INJECT", "context injected via sessionstart")

	out := sessionStartOutput{
		HookSpecificOutput: sessionStartSpecific{
			HookEventName:    "SessionStart",
			AdditionalContext: payload,
		},
	}
	b, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(b)
}
```

**Step 4: Run tests to verify pass**

```bash
go test ./internal/hook/... -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/hook/
git commit -m "feat: add hook/sessionstart — context injection via additionalContext"
```

---

## Task 10: Hook Handler — UserPromptSubmit (Fallback)

**Files:**
- Create: `internal/hook/userpromptsubmit.go`
- Create: `internal/hook/userpromptsubmit_test.go`

**Step 1: Write failing tests**

```go
// internal/hook/userpromptsubmit_test.go
package hook_test

import (
	"strings"
	"testing"

	"github.com/meridian-lex/starfix/internal/hook"
	"github.com/meridian-lex/starfix/internal/state"
)

func TestUserPromptSubmit_NoMarker_EmptyOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)

	output := hook.HandleUserPromptSubmit(hookInput("session-ups-1"), cfg, dir)
	if output != "" {
		t.Errorf("expected empty output when no marker, got: %q", output)
	}
}

func TestUserPromptSubmit_WithMarker_InjectsContext(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ups-2")

	s, _ := state.Load(dir, "session-ups-2")
	s.WriteMarker()

	output := hook.HandleUserPromptSubmit(input, cfg, dir)
	if !strings.Contains(output, "STARFIX") {
		t.Errorf("expected STARFIX header in output, got: %q", output)
	}
}

func TestUserPromptSubmit_WithMarker_DeletesMarker(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ups-3")

	s, _ := state.Load(dir, "session-ups-3")
	s.WriteMarker()

	hook.HandleUserPromptSubmit(input, cfg, dir)

	s2, _ := state.Load(dir, "session-ups-3")
	if s2.MarkerExists() {
		t.Error("marker should be deleted after userpromptsubmit")
	}
}

func TestUserPromptSubmit_ParkFlag_InjectsDirective(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	input := hookInput("session-ups-4")

	s, _ := state.Load(dir, "session-ups-4")
	s.TimeoutFired = true
	s.TimeoutAction = "park"
	s.Save()

	output := hook.HandleUserPromptSubmit(input, cfg, dir)
	if !strings.Contains(output, "park") && !strings.Contains(output, "PARK") {
		t.Errorf("expected park directive in output, got: %q", output)
	}
}
```

**Step 2: Run to verify fail**

```bash
go test ./internal/hook/... 2>&1
```

**Step 3: Implement `internal/hook/userpromptsubmit.go`**

```go
package hook

import (
	"fmt"

	starfixctx "github.com/meridian-lex/starfix/internal/context"
	"github.com/meridian-lex/starfix/internal/config"
	"github.com/meridian-lex/starfix/internal/state"
)

// HandleUserPromptSubmit processes the UserPromptSubmit hook event.
// Acts as fallback if SessionStart did not fire, and handles pending reply/timeout flags.
// Returns plain text to inject into context (stdout), or empty string.
func HandleUserPromptSubmit(input Input, cfg *config.Config, baseDir string) string {
	s, err := state.Load(baseDir, input.SessionID)
	if err != nil {
		return ""
	}

	var payload string

	// Handle pending reply or timeout flags (regardless of marker)
	if s.ReplyReceived {
		payload += fmt.Sprintf("\n--- ADMIRAL REPLY ---\nFleet Admiral replied: %s\n", s.ReplyText)
		s.ReplyReceived = false
		s.ReplyText = ""
		s.EscalationPending = false
		s.Save()
	} else if s.TimeoutFired {
		if s.TimeoutAction == "park" {
			payload += "\n--- STARFIX DIRECTIVE ---\nNo Admiral reply received within timeout. Triage recommended PARK. Please wrap up current work and stop.\n"
		}
		s.TimeoutFired = false
		s.EscalationPending = false
		s.Save()
	}

	// Fallback injection if marker is still present (SessionStart missed it)
	if s.MarkerExists() {
		corePayload := starfixctx.BuildCore(cfg)
		if cfg.ProjectContext && input.CWD != "" {
			if projectLayer := starfixctx.BuildProject(input.CWD); projectLayer != "" {
				corePayload += "\n" + projectLayer
			}
		}
		payload = corePayload + payload
		s.DeleteMarker()
		s.Save()
		logEvent(cfg.LogPath, input.SessionID, "INJECT", "context injected via userpromptsubmit fallback")
	}

	return payload
}
```

**Step 4: Run tests to verify pass**

```bash
go test ./internal/hook/... -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/hook/
git commit -m "feat: add hook/userpromptsubmit — fallback injection and reply/timeout handling"
```

---

## Task 11: Watch-Reply Command

**Files:**
- Create: `internal/hook/watchreply.go`
- Create: `internal/hook/watchreply_test.go`

**Step 1: Write failing tests**

```go
// internal/hook/watchreply_test.go
package hook_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/meridian-lex/starfix/internal/hook"
	"github.com/meridian-lex/starfix/internal/state"
)

func TestWatchReply_ReplyReceived(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.TimeoutSeconds = 2
	cfg.TelegramAdmiralID = 121956871

	// Set up escalation state
	s, _ := state.Load(dir, "session-wr-1")
	s.EscalationPending = true
	s.TriageDefault = "continue"
	s.EscalationSentAt = time.Now().UTC()
	s.Save()

	// Write a reply to the inbound log
	logPath := filepath.Join(dir, "telegram-inbound.log")
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"from":      map[string]interface{}{"id": float64(121956871)},
		"text":      "continue",
	}
	b, _ := json.Marshal(entry)
	os.WriteFile(logPath, append(b, '\n'), 0644)
	cfg.TelegramInboundLog = logPath

	hook.RunWatchReply("session-wr-1", cfg, dir)

	s2, _ := state.Load(dir, "session-wr-1")
	if !s2.ReplyReceived {
		t.Error("ReplyReceived should be true after reply found")
	}
	if s2.ReplyText != "continue" {
		t.Errorf("ReplyText: got %q, want continue", s2.ReplyText)
	}
}

func TestWatchReply_Timeout(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.TimeoutSeconds = 1 // short timeout for test
	cfg.TelegramAdmiralID = 121956871

	logPath := filepath.Join(dir, "empty-inbound.log")
	os.WriteFile(logPath, []byte{}, 0644)
	cfg.TelegramInboundLog = logPath

	s, _ := state.Load(dir, "session-wr-2")
	s.EscalationPending = true
	s.TriageDefault = "park"
	s.EscalationSentAt = time.Now().UTC()
	s.Save()

	hook.RunWatchReply("session-wr-2", cfg, dir)

	s2, _ := state.Load(dir, "session-wr-2")
	if !s2.TimeoutFired {
		t.Error("TimeoutFired should be true after timeout")
	}
	if s2.TimeoutAction != "park" {
		t.Errorf("TimeoutAction: got %q, want park", s2.TimeoutAction)
	}
}
```

Add `cfg.TelegramAdmiralID` field to config (add to `internal/config/config.go` if not already there — it was included in Task 2).

**Step 2: Run to verify fail**

```bash
go test ./internal/hook/... -run TestWatchReply 2>&1
```

**Step 3: Implement `internal/hook/watchreply.go`**

```go
package hook

import (
	"fmt"
	"time"

	"github.com/meridian-lex/starfix/internal/config"
	"github.com/meridian-lex/starfix/internal/state"
	"github.com/meridian-lex/starfix/internal/telegram"
)

// RunWatchReply polls for an Admiral reply until timeout, then executes triage default.
// Intended to run as a background process spawned by HandlePreCompact.
func RunWatchReply(sessionID string, cfg *config.Config, baseDir string) {
	s, err := state.Load(baseDir, sessionID)
	if err != nil {
		return
	}
	if !s.EscalationPending {
		return
	}

	deadline := s.EscalationSentAt.Add(time.Duration(cfg.TimeoutSeconds) * time.Second)
	pollInterval := 5 * time.Second

	for time.Now().Before(deadline) {
		reply, found := telegram.CheckInbound(
			cfg.TelegramInboundLog,
			s.EscalationSentAt,
			cfg.TelegramAdmiralID,
		)
		if found {
			s.ReplyReceived = true
			s.ReplyText = reply
			s.EscalationPending = false
			s.Save()

			if cfg.TelegramEnabled {
				msg := fmt.Sprintf("[Starfix] Reply received from Admiral — %s — session %s",
					reply, shortID(sessionID))
				telegram.Send(cfg.TelegramBinary, msg)
			}
			return
		}
		time.Sleep(pollInterval)
	}

	// Timeout fired
	s.TimeoutFired = true
	s.TimeoutAction = s.TriageDefault
	s.EscalationPending = false
	s.Save()

	if cfg.TelegramEnabled {
		msg := fmt.Sprintf("[Starfix] No reply — proceeding to %s\nSession: %s",
			s.TriageDefault, shortID(sessionID))
		telegram.Send(cfg.TelegramBinary, msg)
	}
}
```

**Step 4: Wire watch-reply into `cmd/starfix/main.go`**

Replace the watch-reply RunE stub:

```go
watchCmd := &cobra.Command{
    Use:   "watch-reply [session_id]",
    Short: "Watch for Telegram reply and execute timeout action",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        cfg, err := config.Load(config.DefaultPath())
        if err != nil {
            return fmt.Errorf("load config: %w", err)
        }
        hook.RunWatchReply(args[0], cfg, state.DefaultBaseDir())
        return nil
    },
}
```

Also wire the hook subcommands with real implementations using `io` and `os` for stdin reading.

**Step 5: Run tests to verify pass**

```bash
go test ./internal/hook/... -v -timeout 30s
```

Expected: all PASS (WatchReply tests take up to ~2s each).

**Step 6: Commit**

```bash
git add internal/hook/ cmd/
git commit -m "feat: add watch-reply command — Telegram reply polling with timeout"
```

---

## Task 12: Wire CLI Subcommands

**Files:**
- Modify: `cmd/starfix/main.go`

**Step 1: Replace all stub RunE handlers in `cmd/starfix/main.go` with full implementations**

Final `cmd/starfix/main.go`:

```go
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/meridian-lex/starfix/internal/config"
	"github.com/meridian-lex/starfix/internal/hook"
	"github.com/meridian-lex/starfix/internal/state"
)

func main() {
	root := &cobra.Command{
		Use:   "starfix",
		Short: "Post-compaction context restoration for Lex",
	}

	hookCmd := &cobra.Command{Use: "hook", Short: "Hook subcommands"}

	hookCmd.AddCommand(
		&cobra.Command{
			Use:   "precompact",
			Short: "Handle PreCompact hook event",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runHook(func(input hook.Input, cfg *config.Config) string {
					hook.HandlePreCompact(input, cfg, state.DefaultBaseDir())
					return ""
				})
			},
		},
		&cobra.Command{
			Use:   "sessionstart",
			Short: "Handle SessionStart hook event",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runHook(func(input hook.Input, cfg *config.Config) string {
					return hook.HandleSessionStart(input, cfg, state.DefaultBaseDir())
				})
			},
		},
		&cobra.Command{
			Use:   "userpromptsubmit",
			Short: "Handle UserPromptSubmit hook event",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runHook(func(input hook.Input, cfg *config.Config) string {
					return hook.HandleUserPromptSubmit(input, cfg, state.DefaultBaseDir())
				})
			},
		},
	)

	watchCmd := &cobra.Command{
		Use:   "watch-reply [session_id]",
		Short: "Watch for Telegram reply and execute timeout action",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.DefaultPath())
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			hook.RunWatchReply(args[0], cfg, state.DefaultBaseDir())
			return nil
		},
	}

	root.AddCommand(hookCmd, watchCmd)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runHook(fn func(hook.Input, *config.Config) string) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	input, err := hook.ReadInput(data)
	if err != nil {
		return fmt.Errorf("parse hook input: %w", err)
	}

	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		// Config missing is non-fatal — use defaults
		cfg = &config.Config{}
	}

	if output := fn(input, cfg); output != "" {
		fmt.Print(output)
	}
	return nil
}
```

**Step 2: Build and smoke-test**

```bash
make build
echo '{"session_id": "test123", "cwd": "/tmp"}' | ./bin/starfix hook precompact
echo '{"session_id": "test123", "cwd": "/tmp"}' | ./bin/starfix hook sessionstart
```

Expected: no errors, no output (no marker present in test123 session).

**Step 3: Run full test suite**

```bash
go test ./... -v
```

Expected: all PASS.

**Step 4: Commit**

```bash
git add cmd/
git commit -m "feat: wire all hook subcommands into CLI — full integration"
```

---

## Task 13: Bash Hook Scripts

**Files:**
- Create: `hooks/precompact.sh`
- Create: `hooks/sessionstart.sh`
- Create: `hooks/userpromptsubmit.sh`

**Step 1: Write `hooks/precompact.sh`**

```bash
#!/usr/bin/env bash
# Starfix — PreCompact hook
# Delegates to starfix binary. Safe to fail silently.
exec "$HOME/.local/bin/starfix" hook precompact
```

**Step 2: Write `hooks/sessionstart.sh`**

```bash
#!/usr/bin/env bash
# Starfix — SessionStart hook
# Outputs additionalContext JSON if post-compaction marker is present.
exec "$HOME/.local/bin/starfix" hook sessionstart
```

**Step 3: Write `hooks/userpromptsubmit.sh`**

```bash
#!/usr/bin/env bash
# Starfix — UserPromptSubmit fallback hook
# Injects context if SessionStart missed the marker.
exec "$HOME/.local/bin/starfix" hook userpromptsubmit
```

**Step 4: Make executable**

```bash
chmod +x hooks/precompact.sh hooks/sessionstart.sh hooks/userpromptsubmit.sh
```

**Step 5: Verify scripts pass stdin through**

```bash
echo '{"session_id": "hooktest", "cwd": "/tmp"}' | bash hooks/precompact.sh
```

Expected: runs without error (binary exists from make build + make install, or from PATH).

**Step 6: Commit**

```bash
git add hooks/
git commit -m "feat: add bash hook wrapper scripts"
```

---

## Task 14: Default Config File

**Files:**
- Create: `config/starfix.cfg`

**Step 1: Write `config/starfix.cfg`**

```yaml
# Starfix Configuration
# Installed to ~/.config/starfix/starfix.cfg
# Edit this file to customise Starfix behaviour.

# Context injection
<!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
project_context: true          # Include CLAUDE.md, README, git log when project detected

# Telegram notifications
telegram_enabled: true
telegram_notify_binary: ~/.local/bin/telegram-notify
telegram_inbound_log: ~/meridian-home/logs/telegram-inbound.log
telegram_admiral_id: 121956871  # Fleet Admiral Lunar Laurus Telegram ID

# Compaction thresholds
summary_threshold: 2           # Send summary on Nth compaction and above
escalation_threshold: 3        # Send escalation with triage on Nth compaction and above

# Escalation timeout
timeout_seconds: 300           # Seconds to wait for Admiral reply before executing triage default

# Logging
log_path: ~/meridian-home/logs/starfix.log

# Lex context paths
memory_path: ~/meridian-home/lex-internal/state/MEMORY.md
task_queue_path: ~/meridian-home/lex-internal/state/TASK-QUEUE.md
state_path: ~/meridian-home/lex-internal/state/STATE.md
```

**Step 2: Commit**

```bash
git add config/starfix.cfg
git commit -m "feat: add default config file"
```

---

## Task 15: install.sh

**Files:**
- Create: `install.sh`
- Create: `uninstall.sh` (thin — calls install.sh remove logic)

**Step 1: Write `install.sh`**

```bash
#!/usr/bin/env bash
# Starfix installer
# Run from the cloned repository root.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_NAME="starfix"
INSTALL_BIN="$HOME/.local/bin"
CONFIG_DIR="$HOME/.config/starfix"
<!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
SETTINGS_FILE="$HOME/.claude/settings.json"

print_banner() {
    echo ""
    echo "  Starfix — Post-Compaction Context Restoration"
    echo ""
}

check_deps() {
    local missing=0
    for dep in go jq; do
        if ! command -v "$dep" &>/dev/null; then
            echo "  ERROR: $dep is required but not found in PATH"
            missing=1
        fi
    done
    if [[ "$missing" -eq 1 ]]; then
        echo ""
        echo "  Note: Go must be in PATH. Try: export PATH=\$PATH:/home/meridian/go/bin"
        exit 1
    fi
}

build_binary() {
    echo "  Building starfix binary..."
    (cd "$SCRIPT_DIR" && go build -o "$SCRIPT_DIR/bin/$BINARY_NAME" ./cmd/starfix/)
    echo "  Build complete."
}

install_files() {
    mkdir -p "$INSTALL_BIN"
    cp "$SCRIPT_DIR/bin/$BINARY_NAME" "$INSTALL_BIN/$BINARY_NAME"
    chmod +x "$INSTALL_BIN/$BINARY_NAME"

    cp "$SCRIPT_DIR/hooks/precompact.sh" "$INSTALL_BIN/starfix-precompact"
    cp "$SCRIPT_DIR/hooks/sessionstart.sh" "$INSTALL_BIN/starfix-sessionstart"
    cp "$SCRIPT_DIR/hooks/userpromptsubmit.sh" "$INSTALL_BIN/starfix-userpromptsubmit"
    chmod +x "$INSTALL_BIN/starfix-precompact" \
             "$INSTALL_BIN/starfix-sessionstart" \
             "$INSTALL_BIN/starfix-userpromptsubmit"

    mkdir -p "$CONFIG_DIR"
    if [[ ! -f "$CONFIG_DIR/starfix.cfg" ]]; then
        cp "$SCRIPT_DIR/config/starfix.cfg" "$CONFIG_DIR/starfix.cfg"
        echo "  Config written to $CONFIG_DIR/starfix.cfg"
    else
        echo "  Existing config preserved at $CONFIG_DIR/starfix.cfg"
    fi

    echo "  Files installed."
}

register_hooks() {
    local tmp
    tmp=$(mktemp)

    jq '
      .PreCompact = (.PreCompact // []) + [{"hooks": [{"type": "command", "command": "'"$INSTALL_BIN/starfix-precompact"'", "timeout": 30}]}] |
      .SessionStart = (.SessionStart // []) + [{"hooks": [{"type": "command", "command": "'"$INSTALL_BIN/starfix-sessionstart"'", "timeout": 15}]}] |
      .UserPromptSubmit = (.UserPromptSubmit // []) + [{"hooks": [{"type": "command", "command": "'"$INSTALL_BIN/starfix-userpromptsubmit"'", "timeout": 15}]}]
    ' "$SETTINGS_FILE" > "$tmp" && mv "$tmp" "$SETTINGS_FILE"

    echo "  Hooks registered in $SETTINGS_FILE"
}

remove_hooks() {
    local tmp
    tmp=$(mktemp)

    jq '
      .PreCompact = [.PreCompact[]? | select(.hooks[0].command | test("starfix") | not)] |
      .SessionStart = [.SessionStart[]? | select(.hooks[0].command | test("starfix") | not)] |
      .UserPromptSubmit = [.UserPromptSubmit[]? | select(.hooks[0].command | test("starfix") | not)]
    ' "$SETTINGS_FILE" > "$tmp" && mv "$tmp" "$SETTINGS_FILE"

    echo "  Hooks removed from $SETTINGS_FILE"
}

remove_files() {
    rm -f "$INSTALL_BIN/$BINARY_NAME" \
          "$INSTALL_BIN/starfix-precompact" \
          "$INSTALL_BIN/starfix-sessionstart" \
          "$INSTALL_BIN/starfix-userpromptsubmit"
    echo "  Binaries removed."

    read -r -p "  Remove config directory $CONFIG_DIR? [y/N] " answer
    if [[ "${answer,,}" == "y" ]]; then
        rm -rf "$CONFIG_DIR"
        echo "  Config removed."
    else
        echo "  Config preserved."
    fi
}

do_install() {
    echo ""
    check_deps
    build_binary
    install_files
    register_hooks
    echo ""
    echo "  Starfix installed. Restart your Lex session to activate hooks."
    echo ""
}

do_remove() {
    echo ""
    remove_hooks
    remove_files
    echo ""
    echo "  Starfix removed. Restart your Lex session to deactivate hooks."
    echo ""
}

# Main menu
print_banner

echo "  1) Install for current user"
echo "  2) Remove"
echo ""
read -r -p "  > " choice

case "$choice" in
    1) do_install ;;
    2) do_remove ;;
    *) echo "  Invalid choice."; exit 1 ;;
esac
```

**Step 2: Make executable**

```bash
chmod +x install.sh
```

**Step 3: Dry-run test (do not actually install yet)**

```bash
bash -n install.sh
```

Expected: no syntax errors.

**Step 4: Commit**

```bash
git add install.sh config/starfix.cfg hooks/
git commit -m "feat: add install.sh — menu-driven installer with hook registration"
```

---

## Task 16: README and Final Integration

**Files:**
- Create: `README.md`
- Create: `.gitignore`

**Step 1: Write `.gitignore`**

```
bin/
*.test
```

**Step 2: Write `README.md`**

```markdown
# Starfix

Post-compaction context restoration for Meridian Lex.

After a context compaction event, Lex loses orientation — active tasks, rank,
directives, project state. Starfix detects compaction and injects critical context
back automatically. It also tracks compaction frequency and escalates to Fleet Admiral
via Telegram when sessions are under sustained pressure.

## Requirements

- Go 1.24+
- jq
- ~/.local/bin/telegram-notify (fleet Telegram binary)

## Install

```bash
git clone https://github.com/Meridian-Lex/Starfix
cd Starfix
./install.sh
```

Select option 1. Restart your Lex session to activate.

## What It Does

| Hook | Trigger | Action |
|------|---------|--------|
| PreCompact | Before compaction | Write marker, track count, send Telegram at threshold |
| SessionStart | After compaction | Inject MEMORY.md + TASK-QUEUE.md + STATE.md + project context |
| UserPromptSubmit | User message | Fallback injection if SessionStart missed; handle reply/timeout flags |

## Telegram Escalation

On the Nth compaction (configurable), Starfix:
1. Runs triage (assesses session pressure)
2. Sends Telegram message with recommendation + timeout default
3. Spawns background reply watcher
4. On reply: follows instruction; on timeout: executes triage default

## Configuration

Edit `~/.config/starfix/starfix.cfg` after install.

## Design

See `docs/plans/2026-02-28-starfix-design.md`.
```

**Step 3: Run full test suite one final time**

```bash
cd ~/meridian-home/projects/Starfix
go test ./... -v
make build
```

Expected: all tests PASS, binary builds cleanly.

**Step 4: Create GitHub repo and push**

```bash
gh repo create Meridian-Lex/Starfix --private --description "Post-compaction context restoration for Meridian Lex"
git remote add origin git@github.com:Meridian-Lex/Starfix.git
git push -u origin master
```

**Step 5: Update projects registry**

Add entry to `~/meridian-home/lex-internal/knowledge/projects.md`:

```markdown
### Starfix
- **Status**: Active
- **Description**: Post-compaction context restoration. Three-hook system (PreCompact, SessionStart, UserPromptSubmit) + Go binary + Telegram escalation.
- **Repo**: https://github.com/Meridian-Lex/Starfix
- **Local path**: `~/meridian-home/projects/Starfix`
- **Go module**: `github.com/meridian-lex/starfix`
```

**Step 6: Final commit**

```bash
git add README.md .gitignore
git commit -m "docs: add README and gitignore — project complete"
```

---

## Parallelization Summary for Subagent Dispatch

```
T1 (scaffold) → must complete first

T2 (config) + T3 (state) → parallel after T1

T4 (context/core) + T5 (context/project) +
T6 (triage) + T7 (telegram) → parallel after T2+T3

T8 (hook/precompact) + T9 (hook/sessionstart) +
T10 (hook/userpromptsubmit) + T11 (watch-reply) → parallel after T4-T7

T12 (bash hooks) → after T8-T11
T13 (install.sh) + T14 (config file) → parallel after T12
T15 (README + final integration) → after all
```

Total estimated tasks: 16
Parallelizable groups: 5
Max parallel agents at peak: 4 (Group 3 and Group 4)
