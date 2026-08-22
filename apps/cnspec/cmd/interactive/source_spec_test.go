// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import "testing"

// The ids used by the curated specs must exist, or a field silently gets no
// values and the user sees an empty box with no explanation.
func TestEverySourceNamedByASpecExists(t *testing.T) {
	for name, spec := range formSpecs {
		for flag, id := range spec.Sources {
			if _, ok := sourceByID(id); !ok {
				t.Errorf("%s --%s names unknown source %q", name, flag, id)
			}
		}
		for flag, id := range spec.LiveSources {
			if _, ok := sourceByID(id); !ok {
				t.Errorf("%s --%s names unknown live source %q", name, flag, id)
			}
		}
		for _, pos := range spec.Positional {
			for _, id := range pos.SourceBy {
				if _, ok := sourceByID(id); id != "" && !ok {
					t.Errorf("%s %q names unknown source %q", name, pos.Label, id)
				}
			}
			for _, id := range pos.LiveSourceBy {
				if _, ok := sourceByID(id); id != "" && !ok {
					t.Errorf("%s %q names unknown live source %q", name, pos.Label, id)
				}
			}
		}
	}
}
