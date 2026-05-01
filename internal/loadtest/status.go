// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/mattn/go-isatty"
)

// StatusReporter prints a periodic summary of progress against Stats. It runs
// in its own goroutine and stops when ctx is cancelled. Output goes to stdout
// directly so the snapshot doesn't get JSON-wrapped by zerolog.
type StatusReporter struct {
	stats    *Stats
	interval time.Duration
	out      io.Writer
	tty      bool
}

// NewStatusReporter wires a reporter against the given stats counter. interval
// of 0 disables periodic printing; the caller can still emit a final summary
// via Final.
func NewStatusReporter(stats *Stats, interval time.Duration) *StatusReporter {
	out := io.Writer(os.Stdout)
	tty := isatty.IsTerminal(os.Stdout.Fd())
	return &StatusReporter{
		stats:    stats,
		interval: interval,
		out:      out,
		tty:      tty,
	}
}

// Run blocks until ctx is cancelled, printing a summary block every interval.
// Designed to be go'd from the main loadtest entry point.
func (r *StatusReporter) Run(ctx context.Context) {
	if r.interval <= 0 {
		return
	}
	tick := time.NewTicker(r.interval)
	defer tick.Stop()

	start := time.Now()
	prevScans := int64(0)
	prevTime := start

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tick.C:
			snapshot := r.snapshot(start, now, prevScans, prevTime)
			r.print(snapshot)
			prevScans = snapshot.totalScans
			prevTime = now
		}
	}
}

// Final prints one last summary covering the entire run. Call from the CLI
// after Run returns so the user sees a closing block even if the last tick
// was a few seconds before completion.
func (r *StatusReporter) Final(start, end time.Time) {
	snap := r.snapshot(start, end, 0, start)
	snap.recentRate = snap.avgRate // no "recent" window for the final view
	r.print(snap)
}

type statusSnapshot struct {
	elapsed       time.Duration
	totalScans    int64
	totalSync     int64
	totalResolve  int64
	totalUpload   int64
	errSync       int64
	errResolve    int64
	errUpload     int64
	assetsHandled int64
	avgRate       float64
	recentRate    float64
}

func (r *StatusReporter) snapshot(start, now time.Time, prevScans int64, prevTime time.Time) statusSnapshot {
	scans := atomic.LoadInt64(&r.stats.ScansSent)
	elapsed := now.Sub(start)
	avgRate := 0.0
	if elapsed > 0 {
		avgRate = float64(scans) / elapsed.Seconds()
	}
	windowSec := now.Sub(prevTime).Seconds()
	recentRate := 0.0
	if windowSec > 0 {
		recentRate = float64(scans-prevScans) / windowSec
	}
	return statusSnapshot{
		elapsed:       elapsed.Truncate(time.Second),
		totalScans:    scans,
		totalSync:     atomic.LoadInt64(&r.stats.SyncCalls),
		totalResolve:  atomic.LoadInt64(&r.stats.ResolveCalls),
		totalUpload:   atomic.LoadInt64(&r.stats.UploadCalls),
		errSync:       atomic.LoadInt64(&r.stats.ErrorsSync),
		errResolve:    atomic.LoadInt64(&r.stats.ErrorsResolve),
		errUpload:     atomic.LoadInt64(&r.stats.ErrorsUpload),
		assetsHandled: atomic.LoadInt64(&r.stats.AssetsHandled),
		avgRate:       avgRate,
		recentRate:    recentRate,
	}
}

func (r *StatusReporter) print(s statusSnapshot) {
	if r.tty {
		r.printPretty(s)
	} else {
		r.printPlain(s)
	}
}

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiCyan   = "\x1b[36m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
)

// formatRate picks a sensible unit so a slow loadtest (≪ 1 scan/s) doesn't
// show up as "0.0/s". Above 1/s we keep seconds; below that we switch to
// per-minute, then per-hour, so the operator sees motion.
func formatRate(perSecond float64) string {
	switch {
	case perSecond >= 1.0:
		return fmt.Sprintf("%.1f/s", perSecond)
	case perSecond*60 >= 1.0:
		return fmt.Sprintf("%.1f/min", perSecond*60)
	default:
		return fmt.Sprintf("%.2f/hr", perSecond*3600)
	}
}

func (r *StatusReporter) printPretty(s statusSnapshot) {
	totalErr := s.errSync + s.errResolve + s.errUpload
	errColor := ansiGreen
	if totalErr > 0 {
		errColor = ansiRed
	}

	header := fmt.Sprintf("%sloadtest%s %s@ %s%s",
		ansiBold+ansiCyan, ansiReset, ansiDim, s.elapsed, ansiReset,
	)
	fmt.Fprintf(r.out, "\n%s ─────────────────────────────────────────\n", header)
	fmt.Fprintf(r.out, "  %sscans%s   %s%-10d%s  %s%s%s  %s(recent %s)%s\n",
		ansiBold, ansiReset,
		ansiBold, s.totalScans, ansiReset,
		ansiYellow, formatRate(s.avgRate), ansiReset,
		ansiDim, formatRate(s.recentRate), ansiReset,
	)
	fmt.Fprintf(r.out, "  sync     %-10d  resolve %-10d  upload %-10d\n",
		s.totalSync, s.totalResolve, s.totalUpload,
	)
	fmt.Fprintf(r.out, "  %serrors%s   sync=%d resolve=%d upload=%d%s\n",
		errColor, ansiReset,
		s.errSync, s.errResolve, s.errUpload, ansiReset,
	)
	fmt.Fprintln(r.out, "")
}

func (r *StatusReporter) printPlain(s statusSnapshot) {
	fmt.Fprintf(r.out,
		"loadtest elapsed=%s scans=%d rate=%s recent=%s sync=%d resolve=%d upload=%d errors[sync=%d resolve=%d upload=%d] assets=%d\n",
		s.elapsed,
		s.totalScans, formatRate(s.avgRate), formatRate(s.recentRate),
		s.totalSync, s.totalResolve, s.totalUpload,
		s.errSync, s.errResolve, s.errUpload,
		s.assetsHandled,
	)
}
