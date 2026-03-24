package triage_test

import (
	"regexp"
	"strings"
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
		CompactionCount:  triage.DefaultContinueBelow,
		TaskQueueContent: "- [in_progress] Implement feature X\n  Clear completion: yes",
	})
	if result.Action != "continue" {
		t.Errorf("Action: got %q, want continue for low count + active task", result.Action)
	}
}

func TestAssess_HighCount_ActiveTask(t *testing.T) {
	result := triage.Assess(triage.Input{
		CompactionCount:  triage.DefaultParkAbove,
		TaskQueueContent: "- [in_progress] Implement feature X",
	})
	if result.Action != "park" {
		t.Errorf("Action: got %q, want park for count>=%d", result.Action, triage.DefaultParkAbove)
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

func TestAssess_CustomThresholds_ParkAbove(t *testing.T) {
	// With ParkAbove=10, count=8 with no active task should continue.
	result := triage.Assess(triage.Input{
		CompactionCount: 8,
		Thresholds:      triage.Thresholds{ParkAbove: 10, ContinueBelow: 2},
	})
	if result.Action != "continue" {
		t.Errorf("Action: got %q, want continue — count 8 below custom ParkAbove 10", result.Action)
	}
}

func TestAssess_CustomThresholds_ContinueBelow(t *testing.T) {
	// With ContinueBelow=5, count=4 with active task should continue.
	result := triage.Assess(triage.Input{
		CompactionCount:  4,
		TaskQueueContent: "- [in_progress] Implement feature X",
		Thresholds:       triage.Thresholds{ParkAbove: 10, ContinueBelow: 5},
	})
	if result.Action != "continue" {
		t.Errorf("Action: got %q, want continue — count 4 within custom ContinueBelow 5", result.Action)
	}
}

func TestAssess_ModeTagInReason(t *testing.T) {
	result := triage.Assess(triage.Input{
		CompactionCount: 6,
		Mode:            "ralph",
	})
	if !strings.Contains(result.Reason, "[ralph]") {
		t.Errorf("Reason %q should contain mode tag [ralph]", result.Reason)
	}
}

func TestAssess_NoModeTag_WhenModeEmpty(t *testing.T) {
	result := triage.Assess(triage.Input{CompactionCount: 0})
	// Check that no bracketed mode token exists (e.g., no [ralph], [autonomous], etc.)
	modePattern := regexp.MustCompile(`\[(?:ralph|autonomous|interactive)\]`)
	if modePattern.MatchString(result.Reason) {
		t.Errorf("Reason %q should not contain bracketed mode tag when Mode is empty", result.Reason)
	}
}

func TestAssess_ZeroThresholds_UseDefaults(t *testing.T) {
	// Zero Thresholds should fall back to DefaultParkAbove and DefaultContinueBelow.
	parkResult := triage.Assess(triage.Input{CompactionCount: triage.DefaultParkAbove})
	if parkResult.Action != "park" {
		t.Errorf("Action: got %q, want park at DefaultParkAbove=%d", parkResult.Action, triage.DefaultParkAbove)
	}
	continueResult := triage.Assess(triage.Input{
		CompactionCount:  triage.DefaultContinueBelow,
		TaskQueueContent: "- [in_progress] Implement feature X",
	})
	if continueResult.Action != "continue" {
		t.Errorf("Action: got %q, want continue at DefaultContinueBelow=%d", continueResult.Action, triage.DefaultContinueBelow)
	}
}
