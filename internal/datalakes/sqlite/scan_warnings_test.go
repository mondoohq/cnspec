// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy/scandb"
	"go.mondoo.com/mql/llx"
)

// fakeCriticalErrorsRuntime satisfies llx.Runtime (via the embedded nil
// interface -- every method but CriticalErrors would panic if called,
// which is the point: writeCriticalErrorsToScanDB must never touch
// anything else on the runtime) plus the optional CriticalErrors() []error
// interface writeCriticalErrorsToScanDB type-asserts for.
type fakeCriticalErrorsRuntime struct {
	llx.Runtime
	errs []error
}

func (f *fakeCriticalErrorsRuntime) CriticalErrors() []error { return f.errs }

// plainRuntime satisfies llx.Runtime but not the CriticalErrors() optional
// interface -- the "mock, or an embedder's own" case the type assertion is
// built to tolerate.
type plainRuntime struct {
	llx.Runtime
}

func newTestStore(t *testing.T) *scandb.SqliteScanDataStore {
	t.Helper()
	store, err := scandb.NewSqliteScanDataStore(filepath.Join(t.TempDir(), "scan.db"), "//assets/1")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store
}

// TestWriteCriticalErrorsToScanDB_PresentWithExpectedJSON is the
// coordinator's "key is present with the expected JSON after a scan with
// critical errors" case, exercised at the exact call WithServices makes --
// after the asset's scan finished, before Finalize.
func TestWriteCriticalErrorsToScanDB_PresentWithExpectedJSON(t *testing.T) {
	store := newTestStore(t)
	runtime := &fakeCriticalErrorsRuntime{errs: []error{
		errors.New("the 'os' provider crashed: connection refused"),
		errors.New("the 'os' provider crashed: connection refused"), // duplicate, must collapse
		errors.New("the 'aws' provider crashed: EOF"),
	}}

	writeCriticalErrorsToScanDB(context.Background(), runtime, store, "//assets/1")

	raw, err := store.GetMetadataByKey(context.Background(), scandb.MetaScanWarnings)
	require.NoError(t, err)

	var got []string
	require.NoError(t, json.Unmarshal([]byte(raw), &got))
	assert.Equal(t, []string{
		"the 'os' provider crashed: connection refused",
		"the 'aws' provider crashed: EOF",
	}, got)
}

// TestWriteCriticalErrorsToScanDB_AbsentOtherwise covers both ways a scan
// can have nothing to report: a runtime that tracked no critical errors,
// and one that doesn't implement CriticalErrors() at all.
func TestWriteCriticalErrorsToScanDB_AbsentOtherwise(t *testing.T) {
	t.Run("runtime with no critical errors", func(t *testing.T) {
		store := newTestStore(t)
		writeCriticalErrorsToScanDB(context.Background(), &fakeCriticalErrorsRuntime{}, store, "//assets/1")

		_, err := store.GetMetadataByKey(context.Background(), scandb.MetaScanWarnings)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("runtime without the optional interface", func(t *testing.T) {
		store := newTestStore(t)
		writeCriticalErrorsToScanDB(context.Background(), &plainRuntime{}, store, "//assets/1")

		_, err := store.GetMetadataByKey(context.Background(), scandb.MetaScanWarnings)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

// TestDedupeAndCapCriticalErrors mirrors policy/executor's
// TestDedupeAndCapScanWarnings for this package's independent copy of the
// same dedup/cap logic.
func TestDedupeAndCapCriticalErrors(t *testing.T) {
	t.Run("empty input returns nil", func(t *testing.T) {
		assert.Nil(t, dedupeAndCapCriticalErrors(nil))
	})

	t.Run("count is capped", func(t *testing.T) {
		errs := make([]error, 0, maxScanWarnings+5)
		for i := 0; i < maxScanWarnings+5; i++ {
			errs = append(errs, errors.New("distinct crash "+string(rune('a'+i%26))))
		}
		assert.Len(t, dedupeAndCapCriticalErrors(errs), maxScanWarnings)
	})

	t.Run("message length is capped", func(t *testing.T) {
		long := strings.Repeat("x", maxScanWarningLen+100)
		out := dedupeAndCapCriticalErrors([]error{errors.New(long)})
		require.Len(t, out, 1)
		assert.Len(t, out[0], maxScanWarningLen)
	})
}
