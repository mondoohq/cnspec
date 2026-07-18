// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

// Package report_conversion turns third-party scanner report files into Mondoo
// FEX/VEX findings (FindingDocument). Each supported format registers a Converter
// under a format name; the `cnspec upload` command and the server console-upload
// path look converters up by name.
//
// This package holds the public, open-standard converters (SARIF, CycloneDX,
// SPDX, JUnit). Vendor/proprietary formats are converted server-side; see
// ADR-062 (docs/adr/062-third-party-data-management.md) for the placement split.
package report_conversion

import (
	"sort"
	"sync"

	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
)

// Converter parses one tool's native report bytes into FEX/VEX documents. It is
// pure: no network, no disk access beyond the input arg.
type Converter func(data []byte) ([]*fex.FindingDocument, error)

// mu guards registry. Registration happens at init() today, but the lock keeps
// the map safe if a converter is ever registered at runtime.
var (
	mu       sync.RWMutex
	registry = map[string]Converter{}
)

// Register wires a format name (the value passed to `--format`) to its converter.
// It is meant to be called from a converter subpackage's init(); registering the
// same format twice panics, which surfaces duplicate wiring at startup.
func Register(format string, c Converter) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[format]; exists {
		panic("report_conversion: duplicate converter registered for format " + format)
	}
	registry[format] = c
}

// Get returns the converter for a format, or (nil, false) if none is registered.
func Get(format string) (Converter, bool) {
	mu.RLock()
	defer mu.RUnlock()
	c, ok := registry[format]
	return c, ok
}

// Formats returns the registered format names, sorted.
func Formats() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
