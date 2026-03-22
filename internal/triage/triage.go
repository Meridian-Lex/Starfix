package triage

import (
	"fmt"
	"strings"
)

// Thresholds holds the park/continue decision boundaries for triage.
// Zero values fall back to built-in defaults.
type Thresholds struct {
	// ParkAbove: compaction count at which the session is unconditionally parked.
	// Default: 5
	ParkAbove int
	// ContinueBelow: compaction count at or below which an in-progress task allows continue.
	// Default: 2
	ContinueBelow int
}

func (t Thresholds) parkAbove() int {
	if t.ParkAbove > 0 {
		return t.ParkAbove
	}
	return 5
}

func (t Thresholds) continueBelow() int {
	if t.ContinueBelow > 0 {
		return t.ContinueBelow
	}
	return 2
}

// Input holds the signals used to assess the session situation.
type Input struct {
	CompactionCount  int
	TaskQueueContent string
	// Mode labels the operational context ("ralph", "autonomous", "interactive").
	// Used for richer reason strings; does not alter decision logic.
	Mode string
	// Thresholds overrides built-in decision boundaries. Zero fields use defaults.
	Thresholds Thresholds
}

// Result holds the triage recommendation.
type Result struct {
	Action string // "continue" or "park"
	Reason string
}

// Assess determines whether Lex should continue or park based on session signals.
func Assess(in Input) Result {
	t := in.Thresholds
	modeTag := ""
	if in.Mode != "" {
		modeTag = " [" + in.Mode + "]"
	}

	if in.CompactionCount >= t.parkAbove() {
		return Result{
			Action: "park",
			Reason: fmt.Sprintf("compaction count is %d%s — session under heavy context pressure", in.CompactionCount, modeTag),
		}
	}

	hasActiveTask := strings.Contains(in.TaskQueueContent, "in_progress")

	if in.CompactionCount <= t.continueBelow() && hasActiveTask {
		return Result{
			Action: "continue",
			Reason: fmt.Sprintf("compaction count is %d%s with an active task in progress — proceeding", in.CompactionCount, modeTag),
		}
	}

	if !hasActiveTask {
		return Result{
			Action: "continue",
			Reason: fmt.Sprintf("no active task in progress%s — safe to continue", modeTag),
		}
	}

	return Result{
		Action: "park",
		Reason: fmt.Sprintf("compaction count is %d%s with active work — recommend parking to avoid context drift", in.CompactionCount, modeTag),
	}
}
