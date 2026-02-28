package triage_test

import (
	"testing"

	"github.com/meridian-lex/starfix/internal/triage"
)

func TestAssess_HighCount_NoTask(t *testing.T) {
	result := triage.Assess(triage.Input{
		CompactionCount:  6,
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
		CompactionCount:  2,
		TaskQueueContent: "- [in_progress] Implement feature X\n  Clear completion: yes",
	})
	if result.Action != "continue" {
		t.Errorf("Action: got %q, want continue for low count + active task", result.Action)
	}
}

func TestAssess_HighCount_ActiveTask(t *testing.T) {
	result := triage.Assess(triage.Input{
		CompactionCount:  5,
		TaskQueueContent: "- [in_progress] Implement feature X",
	})
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
