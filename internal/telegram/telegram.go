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
