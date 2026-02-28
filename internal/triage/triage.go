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

	return Result{
		Action: "park",
		Reason: fmt.Sprintf("compaction count is %d with active work — recommend parking to avoid context drift", in.CompactionCount),
	}
}
