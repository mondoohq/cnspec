// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package backgroundjob

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimeOfDay(t *testing.T) {
	valid := map[string]TimeOfDay{
		"04:00": {Hour: 4},
		"4:00":  {Hour: 4},
		"00:00": {},
		"23:59": {Hour: 23, Minute: 59},
	}
	for in, want := range valid {
		got, err := ParseTimeOfDay(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}

	for _, in := range []string{"", "4", "4am", "24:00", "04:60", "04:00:00", "240"} {
		_, err := ParseTimeOfDay(in)
		assert.Error(t, err, in)
	}
}

func TestTimeOfDay_Next(t *testing.T) {
	at := func(day, hour, minute int) time.Time {
		return time.Date(2026, 10, day, hour, minute, 0, 0, time.UTC)
	}

	tests := []struct {
		name     string
		start    TimeOfDay
		interval time.Duration
		now      time.Time
		want     time.Time
	}{
		{"daily, before the start", TimeOfDay{Hour: 4}, 24 * time.Hour, at(8, 3, 0), at(8, 4, 0)},
		{"daily, exactly at the start", TimeOfDay{Hour: 4}, 24 * time.Hour, at(8, 4, 0), at(9, 4, 0)},
		{"daily, after the start", TimeOfDay{Hour: 4}, 24 * time.Hour, at(8, 5, 0), at(9, 4, 0)},
		{"zero interval is daily", TimeOfDay{Hour: 4}, 0, at(8, 5, 0), at(9, 4, 0)},
		{"interval over a day is daily", TimeOfDay{Hour: 4}, 48 * time.Hour, at(8, 5, 0), at(9, 4, 0)},
		{"every 6h, between slots", TimeOfDay{Hour: 4}, 6 * time.Hour, at(8, 5, 0), at(8, 10, 0)},
		{"every 6h, after the last slot", TimeOfDay{Hour: 4}, 6 * time.Hour, at(8, 23, 0), at(9, 4, 0)},
		// 04:00, 11:00, 18:00, then 01:00 the next morning; 08:00 would be
		// next, but the run restarts at 04:00.
		{"every 7h, spills past midnight", TimeOfDay{Hour: 4}, 7 * time.Hour, at(8, 0, 30), at(8, 1, 0)},
		{"every 7h, restarts at the start", TimeOfDay{Hour: 4}, 7 * time.Hour, at(8, 2, 0), at(8, 4, 0)},
		{"late start, next slot is tomorrow", TimeOfDay{Hour: 23, Minute: 30}, time.Hour, at(8, 23, 45), at(9, 0, 30)},
		{"end of month", TimeOfDay{Hour: 4}, 24 * time.Hour, at(31, 5, 0), time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.start.next(tc.now, tc.interval))
		})
	}
}

// TestTimeOfDay_NextKeepsTheWallClockAcrossDST is the reason slots are built
// with time.Date: adding 24h to 04:00 the day before a daylight saving switch
// would land on 03:00 or 05:00.
func TestTimeOfDay_NextKeepsTheWallClockAcrossDST(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skipf("no time zone data: %v", err)
	}

	// Clocks go forward on 2026-03-29 and back on 2026-10-25.
	got := TimeOfDay{Hour: 4}.next(time.Date(2026, 3, 28, 5, 0, 0, 0, berlin), 24*time.Hour)
	assert.Equal(t, time.Date(2026, 3, 29, 4, 0, 0, 0, berlin), got)
	assert.Equal(t, 4, got.Hour())

	got = TimeOfDay{Hour: 4}.next(time.Date(2026, 10, 24, 5, 0, 0, 0, berlin), 24*time.Hour)
	assert.Equal(t, 4, got.Hour())
	assert.Equal(t, 25, got.Day())

	// Six hours after midnight is 06:00 on the wall clock, even though only
	// five hours pass on the day the clocks go forward.
	got = TimeOfDay{}.next(time.Date(2026, 3, 29, 0, 30, 0, 0, berlin), 6*time.Hour)
	assert.Equal(t, 6, got.Hour())
}

func TestServiceConfig_FirstScan(t *testing.T) {
	now := time.Now()

	cfg := &ServiceConfig{Timer: time.Hour, Splay: time.Hour, FirstScanDelay: 30 * time.Second}
	assert.Equal(t, now.Add(30*time.Second), cfg.firstScan(now), "without a start time the first scan follows the fleet start delay")

	// With a start time the startup scan is skipped, even with no fleet
	// start delay at all.
	start := TimeOfDay{Hour: now.Add(12 * time.Hour).Hour()}
	cfg = &ServiceConfig{Timer: 24 * time.Hour, Splay: 2 * time.Hour, StartTime: &start}
	due := cfg.firstScan(now)
	slot := start.next(now, cfg.Timer)
	assert.False(t, due.Before(slot), "first scan %v is before the start time %v", due, slot)
	assert.True(t, due.Before(slot.Add(2*time.Hour)), "first scan %v is beyond the splay", due)
}

func TestServiceConfig_NextScanStaysWithinTheSplay(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.Local)
	start := TimeOfDay{Hour: 4}
	cfg := &ServiceConfig{Timer: 24 * time.Hour, Splay: 2 * time.Hour, StartTime: &start}
	slot := time.Date(2026, 10, 9, 4, 0, 0, 0, time.Local)

	for i := 0; i < 100; i++ {
		due := cfg.nextScan(now)
		require.False(t, due.Before(slot), "scan at %v is before 04:00", due)
		require.True(t, due.Before(slot.Add(2*time.Hour)), "scan at %v is after 06:00", due)
	}
}

func TestUntilDue(t *testing.T) {
	assert.Equal(t, time.Duration(0), untilDue(time.Now().Add(-time.Hour)), "an overdue scan runs straight away")
	assert.Equal(t, wallClockRecheck, untilDue(time.Now().Add(time.Hour)), "a distant scan is re-checked against the wall clock")

	d := untilDue(time.Now().Add(10 * time.Second))
	assert.Greater(t, d, time.Duration(0))
	assert.LessOrEqual(t, d, 10*time.Second)
}
