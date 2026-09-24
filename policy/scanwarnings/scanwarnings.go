// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package scanwarnings turns recovered provider panics/crashes (mql's
// runtime.CriticalErrors(), e.g. a provider subprocess dying mid-scan) into a
// bounded, deduplicated message list before they are reported to the Mondoo
// Platform and logged (policy/scan.reportCriticalErrors).
package scanwarnings

import "strings"

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
// truncated to at most MaxLen bytes without splitting a multi-byte UTF-8
// character (error text can be localized, e.g. Windows socket errors), and
// the whole list capped to Max entries.
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
			// A byte cut can end mid-rune; ToValidUTF8 drops that trailing
			// partial sequence, so the result stays valid and <= MaxLen bytes.
			msg = strings.ToValidUTF8(msg[:MaxLen], "")
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
