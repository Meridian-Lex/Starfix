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

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
