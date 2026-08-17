// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnspec/v13/cli/progress"
)

// panicProgress is a MultiProgress whose Open panics, standing in for a bug in
// the progress renderer.
type panicProgress struct {
	progress.NoopMultiProgress
}

func (panicProgress) Open() error { panic("progress bar exploded") }

// TestRunProgressBar_ContainsPanic verifies that a panic while opening the
// progress bar is contained and does not propagate, so a UI bug can't crash the
// scan process.
func TestRunProgressBar_ContainsPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		runProgressBar(panicProgress{})
	})
}

// TestRunProgressBar_NoPanicPath ensures the normal path is a clean pass-through.
func TestRunProgressBar_NoPanicPath(t *testing.T) {
	assert.NotPanics(t, func() {
		runProgressBar(progress.NoopMultiProgress{})
	})
}
