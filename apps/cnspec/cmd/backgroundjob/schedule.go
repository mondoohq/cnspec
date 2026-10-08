// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package backgroundjob

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/cockroachdb/errors"
)

// TimeOfDay is a wall-clock time, to the minute, in the host's local time
// zone. It anchors the scan schedule when an operator wants scans inside a
// fixed window, such as 04:00 to 06:00, rather than every hour from whenever
// the service happened to start.
type TimeOfDay struct {
	Hour   int
	Minute int
}

// ParseTimeOfDay reads a 24-hour "HH:MM" time such as "04:00" or "4:00".
func ParseTimeOfDay(s string) (TimeOfDay, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return TimeOfDay{}, errors.Newf("invalid start time %q, expected a 24-hour HH:MM time such as \"04:00\"", s)
	}
	return TimeOfDay{Hour: t.Hour(), Minute: t.Minute()}, nil
}

func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

// next returns the first slot strictly after now. Slots fall at t and then
// every interval after it, and the run of slots restarts at t each day, so an
// interval that does not divide the day evenly leaves a shorter gap before t.
// An interval of zero, or of a day or longer, gives one slot a day.
//
// Slots are built with time.Date in now's location rather than by adding
// durations, so they stay on the wall clock across daylight saving changes:
// a 04:00 slot is 04:00 local time on both sides of the switch.
func (t TimeOfDay) next(now time.Time, interval time.Duration) time.Time {
	step := int(interval / time.Minute)
	if step >= 24*60 {
		step = 0
	}

	y, m, d := now.Date()
	loc := now.Location()
	// Start from yesterday's run: with an interval that does not divide the
	// day, it spills past midnight into today.
	for day := d - 1; ; day++ {
		end := time.Date(y, m, day+1, t.Hour, t.Minute, 0, 0, loc)
		for minute := t.Minute; ; minute += step {
			slot := time.Date(y, m, day, t.Hour, minute, 0, 0, loc)
			if !slot.Before(end) {
				break
			}
			if slot.After(now) {
				return slot
			}
			if step <= 0 {
				break
			}
		}
	}
}

// firstScan is when the first scan after the service starts is due.
func (cfg *ServiceConfig) firstScan(now time.Time) time.Time {
	if cfg.StartTime != nil {
		// No scan at startup. The service restarts on every reboot and
		// upgrade, and a scan each time would land outside the window the
		// operator asked for.
		return cfg.nextScan(now)
	}
	return now.Add(cfg.FirstScanDelay)
}

// nextScan is when the next scan is due, given that the previous one
// finished at now.
func (cfg *ServiceConfig) nextScan(now time.Time) time.Time {
	var splay time.Duration
	if cfg.Splay > 0 {
		splay = time.Duration(rand.Int63n(int64(cfg.Splay)))
	}
	if cfg.StartTime != nil {
		return cfg.StartTime.next(now, cfg.Timer).Add(splay)
	}
	return now.Add(cfg.Timer + splay)
}

// wallClockRecheck caps how long the service sleeps in one go before looking
// at the clock again. Go timers run on the monotonic clock, which stands still
// while the host is suspended, so a single long timer set before a laptop
// slept through 04:00 would go off hours after it woke up.
const wallClockRecheck = time.Minute

// untilDue is how long to sleep before checking whether due has arrived.
//
// For a schedule anchored to a start time, due comes from time.Date and has no
// monotonic reading, so it is compared on the wall clock and a scan missed
// during suspend runs as soon as the host wakes. For a plain interval, due is
// time.Now plus a duration and keeps its monotonic reading, so it is compared
// exactly as a single timer would have been.
func untilDue(due time.Time) time.Duration {
	return min(max(time.Until(due), 0), wallClockRecheck)
}

// describeSchedule is the startup log line that tells an operator when their
// scans are going to run.
func (cfg *ServiceConfig) describeSchedule() string {
	splay := int(cfg.Splay.Minutes())
	if cfg.StartTime == nil {
		return fmt.Sprintf("scan interval is %d minute(s) with a splay of %d minute(s)",
			int(cfg.Timer.Minutes()), splay)
	}
	zone, _ := time.Now().Zone()
	if cfg.Timer <= 0 || cfg.Timer >= 24*time.Hour {
		return fmt.Sprintf("scans run daily at %s %s with a splay of %d minute(s)",
			cfg.StartTime, zone, splay)
	}
	return fmt.Sprintf("scans run every %d minute(s) from %s %s, restarting at %s each day, with a splay of %d minute(s)",
		int(cfg.Timer.Minutes()), cfg.StartTime, zone, cfg.StartTime, splay)
}
