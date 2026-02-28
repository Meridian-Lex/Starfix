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
