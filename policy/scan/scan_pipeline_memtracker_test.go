// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanDispatcher_InFlightReflectsHeldScanSlots(t *testing.T) {
	d := &scanDispatcher{scanSem: make(chan struct{}, 4)}
	require.Equal(t, 0, d.inFlight())

	d.scanSem <- struct{}{}
	d.scanSem <- struct{}{}
	require.Equal(t, 2, d.inFlight())

	<-d.scanSem
	require.Equal(t, 1, d.inFlight())
}

func TestScanDispatcher_InFlightZeroWhenNoSemaphore(t *testing.T) {
	d := &scanDispatcher{}
	require.Equal(t, 0, d.inFlight())
}
