// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
)

func TestExecutionManagerRecoversPanic(t *testing.T) {
	runQueue := make(chan runQueueItem, 1)
	resultChan := make(chan *llx.RawResult, 1)
	em := newExecutionManager(nil, runQueue, resultChan, time.Second, false)
	em.Start()

	// A nil code bundle panics inside executeCodeBundle. The manager must
	// recover and surface it as an unrecoverable error instead of crashing.
	runQueue <- runQueueItem{}

	select {
	case err := <-em.Err():
		require.ErrorContains(t, err, "panic during query execution")
	case <-time.After(5 * time.Second):
		t.Fatal("expected the execution manager to report the panic as an error")
	}

	em.Stop()
}
