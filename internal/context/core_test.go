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
	payload := starfixctx.BuildCore(cfg)
	_ = payload
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	os.WriteFile(path, []byte(content), 0644)
}
