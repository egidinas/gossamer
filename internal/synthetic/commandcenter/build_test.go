package commandcenter

import (
	"strings"
	"testing"
	"time"
)

func TestCommandCenterOperatorNextBeforeBreakdownWindow(t *testing.T) {
	breakdownStart := time.Date(2026, 1, 16, 8, 30, 0, 0, time.UTC)
	breakdownEnd := breakdownStart.Add(3 * time.Hour)
	resetEnd := breakdownEnd.Add(3 * time.Hour)
	fixedTime := breakdownStart.Add(-2 * time.Hour)

	got := commandCenterOperatorNext(fixedTime, "complete", breakdownStart, breakdownEnd, resetEnd)
	if strings.Contains(got, "breakdown in progress") {
		t.Fatalf("operator next = %q, should not report breakdown before window starts", got)
	}
	if !strings.Contains(got, "prepare breakdown slot") {
		t.Fatalf("operator next = %q, want prepare breakdown slot", got)
	}
}

func TestCommandCenterOperatorNextDuringBreakdownWindow(t *testing.T) {
	breakdownStart := time.Date(2026, 1, 16, 8, 30, 0, 0, time.UTC)
	breakdownEnd := breakdownStart.Add(3 * time.Hour)
	resetEnd := breakdownEnd.Add(3 * time.Hour)
	fixedTime := breakdownStart.Add(30 * time.Minute)

	got := commandCenterOperatorNext(fixedTime, "complete", breakdownStart, breakdownEnd, resetEnd)
	if got != "breakdown in progress" {
		t.Fatalf("operator next = %q, want breakdown in progress", got)
	}
}
