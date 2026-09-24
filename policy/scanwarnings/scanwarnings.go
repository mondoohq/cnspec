// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package scanwarnings holds the dedupe/cap logic shared by every path that
// turns recovered provider panics/crashes (mql's runtime.CriticalErrors(),
// e.g. a provider subprocess dying mid-scan) into a bounded, reportable
// message list:
//
//   - the terminal/JSON report (policy/scan.reportCriticalErrors)
//   - the upstream StoreResultsReq.scan_warnings field
//     (policy/executor.ExecuteResolvedPolicy)
//   - the --output-scan-db scan database
//     (internal/datalakes/sqlite.writeCriticalErrorsToScanDB)
//
// All three need the same caps, or one path can carry more or longer
// warnings than the others for the same asset.
package scanwarnings

const (
	// Max caps how many distinct crash messages are attached to a report,
	// so a crash storm on one asset can't bloat it.
	Max = 20
	// MaxLen caps each message's length in bytes.
	MaxLen = 1024
)

// DedupeAndCap converts recovered-panic errors into a deduplicated,
// size-capped message list. Nil errors are skipped. Messages are
// deduplicated by their exact string, in the order first seen, each
// truncated to MaxLen bytes, and the whole list capped to Max entries.
//
// mql's own dedup (collapsing repeated failures on one crashed provider
// into a single CriticalErrors() entry) cannot be assumed here: cnspec pins
// mql versions independently, so an older mql build may not have it.
func DedupeAndCap(errs []error) []string {
	if len(errs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(errs))
	out := make([]string, 0, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		msg := err.Error()
		if len(msg) > MaxLen {
			msg = msg[:MaxLen]
		}
		if _, dup := seen[msg]; dup {
			continue
		}
		seen[msg] = struct{}{}
		out = append(out, msg)
		if len(out) >= Max {
			break
		}
	}
	return out
}
