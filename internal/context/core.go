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
