package commandcenter

import (
	"time"

	"github.com/egidinas/gossamer/internal/contracts"
)

func commandCenterResetDuration(laneIndex, runIndex int) time.Duration {
	hours := []int{18, 22, 26, 20, 24, 28}
	return time.Duration(hours[(laneIndex+runIndex)%len(hours)]) * time.Hour
}

func commandCenterPreferredStartHour(laneIndex, runIndex int) int {
	slots := []int{11, 13, 15, 17}
	return slots[(laneIndex+runIndex)%len(slots)]
}

func commandCenterTestStartAfterPrep(ready time.Time, laneIndex, runIndex int) time.Time {
	prepStart := commandCenterPrepStart(ready, laneIndex, runIndex)
	return AddBusinessDuration(prepStart, 3*time.Hour)
}

func commandCenterPrepStart(t time.Time, laneIndex, runIndex int) time.Time {
	hour := commandCenterPreferredStartHour(laneIndex, runIndex)
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return commandCenterEarliestPrep(nextWorkdayAt(t, hour).Add(-3 * time.Hour))
	}
	slot := time.Date(t.Year(), t.Month(), t.Day(), hour-3, 0, 0, 0, time.UTC)
	slot = commandCenterEarliestPrep(slot)
	if !slot.Before(t) && IsBusinessHour(slot) {
		return slot
	}
	for _, lateHour := range []int{hour + 2, hour + 4} {
		if lateHour > 16 {
			continue
		}
		lateSlot := time.Date(t.Year(), t.Month(), t.Day(), lateHour-3, 0, 0, 0, time.UTC)
		lateSlot = commandCenterEarliestPrep(lateSlot)
		if !lateSlot.Before(t) && IsBusinessHour(lateSlot) {
			return lateSlot
		}
	}
	return commandCenterEarliestPrep(nextWorkdayAt(t.AddDate(0, 0, 1), hour).Add(-3 * time.Hour))
}

func commandCenterEarliestPrep(t time.Time) time.Time {
	earliest := time.Date(t.Year(), t.Month(), t.Day(), 8, 30, 0, 0, time.UTC)
	if t.Before(earliest) {
		return earliest
	}
	return t
}

func commandCenterDeconflictStart(start time.Time, runDuration time.Duration, laneIndex, runIndex int, assignedStarts, assignedFinishes, assignedBreakdowns, assignedResets []time.Time) time.Time {
	candidate := commandCenterBusinessTestStart(start)
	for attempts := 0; attempts < 128; attempts++ {
		finish := candidate.Add(runDuration)
		breakdownStart, _, resetStart, _ := commandCenterOperatorWindows(finish, laneIndex, runIndex)
		if commandCenterSpaced(candidate, assignedStarts, 90*time.Minute) &&
			commandCenterSpaced(finish, assignedFinishes, 90*time.Minute) &&
			commandCenterDailyRoom(candidate, assignedStarts, 2) &&
			commandCenterDailyRoom(finish, assignedFinishes, 2) &&
			commandCenterDailyRoom(breakdownStart, assignedBreakdowns, 2) &&
			commandCenterDailyRoom(resetStart, assignedResets, 2) {
			return candidate
		}
		candidate = commandCenterBusinessTestStart(AddBusinessDuration(candidate, 90*time.Minute))
	}
	return candidate
}

func commandCenterBusinessTestStart(t time.Time) time.Time {
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return nextWorkdayAt(t, 11).Add(30 * time.Minute)
	}
	earliest := time.Date(t.Year(), t.Month(), t.Day(), 11, 30, 0, 0, time.UTC)
	if t.Before(earliest) {
		return earliest
	}
	end := time.Date(t.Year(), t.Month(), t.Day(), 18, 0, 0, 0, time.UTC)
	if !t.Before(end) {
		return nextWorkdayAt(t.AddDate(0, 0, 1), 11).Add(30 * time.Minute)
	}
	return t
}

func commandCenterSpaced(candidate time.Time, assigned []time.Time, minGap time.Duration) bool {
	for _, other := range assigned {
		gap := candidate.Sub(other)
		if gap < 0 {
			gap = -gap
		}
		if gap < minGap {
			return false
		}
	}
	return true
}

func commandCenterDailyRoom(candidate time.Time, assigned []time.Time, maxPerDay int) bool {
	day := candidate.Format("2006-01-02")
	count := 0
	for _, other := range assigned {
		if other.Format("2006-01-02") == day {
			count++
		}
	}
	return count < maxPerDay
}

func commandCenterOperatorWindows(runEnd time.Time, laneIndex, runIndex int) (time.Time, time.Time, time.Time, time.Time) {
	start := workdayResetStart(runEnd)
	jitter := time.Duration((laneIndex*17+runIndex*11)%4) * 30 * time.Minute
	breakdownStart := AddBusinessDuration(start, jitter)
	breakdownEnd := AddBusinessDuration(breakdownStart, 3*time.Hour)
	resetStart := breakdownEnd
	resetEnd := AddBusinessDuration(resetStart, 3*time.Hour)
	return breakdownStart, breakdownEnd, resetStart, resetEnd
}

// AddBusinessDuration advances start by duration, skipping non-business hours.
func AddBusinessDuration(start time.Time, duration time.Duration) time.Time {
	if duration <= 0 {
		return start
	}
	current := start
	remaining := duration
	for remaining > 0 {
		current = nextBusinessTime(current)
		dayEnd := time.Date(current.Year(), current.Month(), current.Day(), 18, 0, 0, 0, time.UTC)
		available := dayEnd.Sub(current)
		if remaining <= available {
			return current.Add(remaining)
		}
		remaining -= available
		current = nextWorkdayAt(current.AddDate(0, 0, 1), 8)
	}
	return current
}

func nextBusinessTime(t time.Time) time.Time {
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return nextWorkdayAt(t, 8)
	}
	start := time.Date(t.Year(), t.Month(), t.Day(), 8, 0, 0, 0, time.UTC)
	if t.Before(start) {
		return start
	}
	end := time.Date(t.Year(), t.Month(), t.Day(), 18, 0, 0, 0, time.UTC)
	if !t.Before(end) {
		return nextWorkdayAt(t.AddDate(0, 0, 1), 8)
	}
	return t
}

// IsBusinessHour reports whether t falls within business hours (Mon–Fri 08:00–18:00 UTC).
func IsBusinessHour(t time.Time) bool {
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	start := time.Date(t.Year(), t.Month(), t.Day(), 8, 0, 0, 0, time.UTC)
	end := time.Date(t.Year(), t.Month(), t.Day(), 18, 0, 0, 0, time.UTC)
	return !t.Before(start) && t.Before(end)
}

func weekendBands(start, end time.Time) []contracts.CommandCenterBand {
	bands := []contracts.CommandCenterBand{}
	cursor := start
	for cursor.Before(end) {
		if cursor.Weekday() == time.Saturday {
			bandEnd := cursor.AddDate(0, 0, 2)
			if bandEnd.After(end) {
				bandEnd = end
			}
			bands = append(bands, contracts.CommandCenterBand{ID: "weekend-" + cursor.Format("20060102"), Label: "Weekend", Kind: "weekend", Start: cursor.Format(time.RFC3339), End: bandEnd.Format(time.RFC3339)})
			cursor = bandEnd
			continue
		}
		cursor = cursor.AddDate(0, 0, 1)
	}
	return bands
}

func workdayAt(day time.Time, hour int) time.Time {
	t := time.Date(day.Year(), day.Month(), day.Day(), hour, 0, 0, 0, time.UTC)
	for t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		t = t.AddDate(0, 0, 1)
	}
	return t
}

func nextWorkdayAt(t time.Time, hour int) time.Time {
	next := workdayAt(t, hour)
	if !next.Before(t) {
		return next
	}
	return workdayAt(t.AddDate(0, 0, 1), hour)
}

func workdayResetStart(t time.Time) time.Time {
	next := time.Date(t.Year(), t.Month(), t.Day(), 8, 0, 0, 0, time.UTC)
	if t.Hour() >= 8 {
		next = next.AddDate(0, 0, 1)
	}
	return nextWorkdayAt(next, 8)
}
