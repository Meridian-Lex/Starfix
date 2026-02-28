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
 // IDENTITY-EXCEPTION: functional internal reference — not for public exposure
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
