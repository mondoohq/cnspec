// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import "strings"

// The launcher's list holds connectors and nothing else. Every row leads to a
// command; a row that leads nowhere is a dead end in a UI built around running
// things, which is why the AI-agent skills that used to sit here were removed.

func matchesTokens(haystack string, tokens []string) bool {
	for _, t := range tokens {
		if !strings.Contains(haystack, t) {
			return false
		}
	}
	return true
}
