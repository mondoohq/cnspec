# Scan Statistics on `ReportUploadCompleted` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Attach per-scan statistics (scan duration, queries executed, queries errored, upload size) to the `ReportUploadCompleted` RPC cnspec makes at the end of a scan upload, via an extensible `google.protobuf.Any` payload carrying a flat, namespaced metric list.

**Architecture:** Two levels. Level 1: a `google.protobuf.Any details` field on the generic `ReportUploadCompletedReq` keeps the base message decoupled from scans — scan is one upload kind among several. Level 2: the `Any` packs a `ScanStatistics { repeated Metric }` message where every metric is a namespaced name + unit + `oneof` value, so new metrics (including future provider-contributed ones under `provider.<name>.*`) never require a proto change. A small `scanstats.Collector` accumulates metrics and is plumbed through the existing SQLite upload path.

**Tech Stack:** Go, protobuf (proto3), protoc-gen-go + protoc-gen-rangerrpc + protoc-gen-go-vtproto, SQLite scan data store, ranger-rpc.

---

## Design references

- Spec: `docs/superpowers/specs/2026-07-17-scan-statistics-report-upload-design.md`
- Base RPC message: `policy/cnspec_policy.proto` (`ReportUploadCompletedReq`, currently `upload_session_id`, `scope_mrn`).
- Call site: `internal/datalakes/sqlite/sqlite.go` — `WithServices(...)` (times/creates), `uploadScanDataStore(...)` (attaches). Runs only when `upstream != nil`.
- Counting seam: `policy/scandb/scan_data_store_wrapper.go` — `WriteScore` / `WriteData`.
- Error identification: `score.Type == policy.ScoreType_Error` (`policy/score.go:14`); data result errored when `result.GetError() != ""` (`mql/llx/llx.pb.go:842`).
- Codegen: `go generate ./policy` (directive `policy/policy.go:23`; needs the `mql` symlink present, which it is).

## Namespacing / well-known metric names (v1)

| Metric name | unit | source |
|---|---|---|
| `cnspec.scan.duration` | `ms` | wall time around the scan closure `f(ctx, ls)` |
| `cnspec.scan.queries_executed` | `count` | scores + data results written to the scan db |
| `cnspec.scan.queries_errored` | `count` | error-typed scores + data results with non-empty error |
| `cnspec.scan.upload_size` | `bytes` | `os.Stat(scanDataPath).Size()` after `Finalize()` |

Provider metrics (future, not in v1) use `provider.<name>.*` via the same Collector adders.

---

## Task 1: Proto — add `Any details` + `ScanStatistics`/`Metric`, regenerate, prove round-trip

This task is also the de-risk for the `Any` pattern (new to this repo). Do it first.

**Files:**
- Modify: `policy/cnspec_policy.proto` (imports; `ReportUploadCompletedReq`; new messages)
- Regenerate: `policy/cnspec_policy.pb.go`, `policy/cnspec_policy_vtproto.pb.go`, `policy/cnspec_policy.ranger.go` (via `go generate ./policy`)
- Test: `policy/report_upload_stats_test.go` (new, package `policy`)

- [ ] **Step 1: Add the `any.proto` import**

In `policy/cnspec_policy.proto`, in the import block near the top (after the existing `import "google/protobuf/timestamp.proto";`), add:

```protobuf
import "google/protobuf/any.proto";
```

- [ ] **Step 2: Add the `details` field to `ReportUploadCompletedReq`**

Replace the existing message:

```protobuf
message ReportUploadCompletedReq {
  string upload_session_id = 1;
  string scope_mrn = 2;
}
```

with:

```protobuf
message ReportUploadCompletedReq {
  string upload_session_id = 1;
  string scope_mrn = 2;
  // details carries an optional, per-upload-kind completion payload. For a
  // scan-database upload this packs a ScanStatistics. New upload kinds can pack
  // their own message without changing this message.
  google.protobuf.Any details = 3;
}
```

- [ ] **Step 3: Add the `ScanStatistics` and `Metric` messages**

Add these two messages immediately after `ReportUploadCompletedReq`:

```protobuf
// ScanStatistics is the completion payload for a scan-database upload. Every
// statistic is a namespaced Metric so new metrics require no proto change.
message ScanStatistics {
  repeated Metric metrics = 1;
}

// Metric is a single namespaced measurement. name uses a dotted namespace
// (e.g. "cnspec.scan.duration", "provider.aws.api_calls"); unit is an optional
// rendering hint (e.g. "ms", "bytes", "count").
message Metric {
  string name = 1;
  string unit = 2;
  oneof value {
    int64 int_value = 3;
    double double_value = 4;
    bool bool_value = 5;
    string string_value = 6;
  }
}
```

- [ ] **Step 4: Regenerate**

Run: `go generate ./policy`
Expected: completes with no error; `git status` shows modified `policy/cnspec_policy.pb.go`, `policy/cnspec_policy_vtproto.pb.go`, `policy/cnspec_policy.ranger.go`.

If codegen fails on the `any.proto` import path, verify the `mql` symlink exists (`ls -l mql`) and re-run; the protoc `--proto_path` already includes the well-known types via `protoc-gen-go`.

- [ ] **Step 5: Confirm the generated Go compiles**

Run: `go build ./policy/...`
Expected: success. Confirms vtproto generated valid `Any` marshaling (via the planetscale `anypb` VT wrapper).

- [ ] **Step 6: Write the failing round-trip test**

Create `policy/report_upload_stats_test.go`:

```go
package policy

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func newReqWithStats(t *testing.T) *ReportUploadCompletedReq {
	t.Helper()
	stats := &ScanStatistics{Metrics: []*Metric{
		{Name: "cnspec.scan.duration", Unit: "ms", Value: &Metric_IntValue{IntValue: 4200}},
		{Name: "cnspec.scan.queries_executed", Unit: "count", Value: &Metric_IntValue{IntValue: 128}},
	}}
	details, err := anypb.New(stats)
	require.NoError(t, err)
	return &ReportUploadCompletedReq{
		UploadSessionId: "session-1",
		ScopeMrn:        "//assets/1",
		Details:         details,
	}
}

func requireStatsRoundTrip(t *testing.T, got *ReportUploadCompletedReq) {
	t.Helper()
	require.Equal(t, "session-1", got.UploadSessionId)
	require.NotNil(t, got.Details)
	var stats ScanStatistics
	require.NoError(t, got.Details.UnmarshalTo(&stats))
	require.Len(t, stats.Metrics, 2)
	require.Equal(t, "cnspec.scan.duration", stats.Metrics[0].Name)
	require.Equal(t, int64(4200), stats.Metrics[0].GetIntValue())
}

func TestReportUploadCompletedReq_AnyRoundTrip_Proto(t *testing.T) {
	req := newReqWithStats(t)
	raw, err := proto.Marshal(req)
	require.NoError(t, err)
	var got ReportUploadCompletedReq
	require.NoError(t, proto.Unmarshal(raw, &got))
	requireStatsRoundTrip(t, &got)
}

func TestReportUploadCompletedReq_AnyRoundTrip_VT(t *testing.T) {
	req := newReqWithStats(t)
	raw, err := req.MarshalVT()
	require.NoError(t, err)
	var got ReportUploadCompletedReq
	require.NoError(t, got.UnmarshalVT(raw))
	requireStatsRoundTrip(t, &got)
}
```

- [ ] **Step 7: Run the tests**

Run: `go test ./policy/ -run TestReportUploadCompletedReq_AnyRoundTrip -v`
Expected: both PASS. This proves the `Any` payload survives both stock proto and vtproto marshaling — the de-risk is complete.

- [ ] **Step 8: Commit**

```bash
git add policy/cnspec_policy.proto policy/cnspec_policy.pb.go policy/cnspec_policy_vtproto.pb.go policy/cnspec_policy.ranger.go policy/report_upload_stats_test.go
git commit -m "✨ policy: add Any details payload + ScanStatistics to ReportUploadCompleted"
```

---

## Task 2: `scanstats` package — Collector, constants, ToProto

**Files:**
- Create: `policy/scanstats/collector.go`
- Test: `policy/scanstats/collector_test.go`

- [ ] **Step 1: Write the failing test**

Create `policy/scanstats/collector_test.go`:

```go
package scanstats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCollector_Empty_ToProtoNil(t *testing.T) {
	var c *Collector
	require.Nil(t, c.ToProto())      // nil receiver is safe
	require.Nil(t, New().ToProto())  // no metrics -> nil
}

func TestCollector_Adders(t *testing.T) {
	c := New()
	c.AddDuration(MetricScanDuration, 4200*time.Millisecond)
	c.AddInt(MetricQueriesExecuted, "count", 128)
	c.AddDouble("cnspec.scan.avg_latency", "ms", 3.5)
	c.AddBool("cnspec.scan.truncated", true)

	stats := c.ToProto()
	require.Len(t, stats.Metrics, 4)

	require.Equal(t, MetricScanDuration, stats.Metrics[0].Name)
	require.Equal(t, "ms", stats.Metrics[0].Unit)
	require.Equal(t, int64(4200), stats.Metrics[0].GetIntValue())

	require.Equal(t, int64(128), stats.Metrics[1].GetIntValue())
	require.Equal(t, "count", stats.Metrics[1].Unit)

	require.Equal(t, 3.5, stats.Metrics[2].GetDoubleValue())
	require.Equal(t, true, stats.Metrics[3].GetBoolValue())
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./policy/scanstats/ -v`
Expected: FAIL — package/`New`/`Collector` not defined.

- [ ] **Step 3: Write the implementation**

Create `policy/scanstats/collector.go`:

```go
// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

// Package scanstats collects namespaced metrics about a scan and converts them
// into the policy.ScanStatistics proto attached to ReportUploadCompleted.
package scanstats

import (
	"sync"
	"time"

	"go.mondoo.com/cnspec/v13/policy"
)

// Well-known core metric names. Core cnspec metrics use the "cnspec.scan.*"
// namespace; provider-contributed metrics use "provider.<name>.*".
const (
	MetricScanDuration    = "cnspec.scan.duration"          // unit: ms
	MetricQueriesExecuted = "cnspec.scan.queries_executed"  // unit: count
	MetricQueriesErrored  = "cnspec.scan.queries_errored"   // unit: count
	MetricUploadSize      = "cnspec.scan.upload_size"       // unit: bytes
)

// Collector accumulates scan metrics. It is safe for concurrent use.
type Collector struct {
	mu      sync.Mutex
	metrics []*policy.Metric
}

// New returns an empty Collector.
func New() *Collector { return &Collector{} }

func (c *Collector) add(m *policy.Metric) {
	c.mu.Lock()
	c.metrics = append(c.metrics, m)
	c.mu.Unlock()
}

// AddInt records an integer metric (counts, byte sizes, unix timestamps).
func (c *Collector) AddInt(name, unit string, v int64) {
	c.add(&policy.Metric{Name: name, Unit: unit, Value: &policy.Metric_IntValue{IntValue: v}})
}

// AddDuration records a duration as an integer number of milliseconds.
func (c *Collector) AddDuration(name string, d time.Duration) {
	c.AddInt(name, "ms", d.Milliseconds())
}

// AddDouble records a floating-point metric (ratios, averages).
func (c *Collector) AddDouble(name, unit string, v float64) {
	c.add(&policy.Metric{Name: name, Unit: unit, Value: &policy.Metric_DoubleValue{DoubleValue: v}})
}

// AddBool records a boolean flag metric.
func (c *Collector) AddBool(name string, v bool) {
	c.add(&policy.Metric{Name: name, Value: &policy.Metric_BoolValue{BoolValue: v}})
}

// ToProto returns the collected metrics as a ScanStatistics, or nil when the
// collector is nil or has no metrics.
func (c *Collector) ToProto() *policy.ScanStatistics {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.metrics) == 0 {
		return nil
	}
	return &policy.ScanStatistics{Metrics: c.metrics}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./policy/scanstats/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add policy/scanstats/collector.go policy/scanstats/collector_test.go
git commit -m "✨ scanstats: add Collector for namespaced scan metrics"
```

---

## Task 3: Count executed / errored queries in `ScanDataStoreWrapper`

**Files:**
- Modify: `policy/scandb/scan_data_store_wrapper.go` (add counters + getters)
- Test: `policy/scandb/scan_data_store_wrapper_stats_test.go` (new)

- [ ] **Step 1: Write the failing test**

Create `policy/scandb/scan_data_store_wrapper_stats_test.go`:

```go
package scandb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/llx"
)

// countingStore is a minimal ScanDataStore that only needs Write* for this test.
// It embeds a real store via a temp sqlite db so Write* succeed.
func newWrapperForTest(t *testing.T) *ScanDataStoreWrapper {
	t.Helper()
	store, err := NewSqliteScanDataStore(t.TempDir()+"/scan.db", "//assets/1")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return NewScanDataStoreWrapper(store, "//assets/1")
}

func TestWrapper_CountsExecutedAndErrored(t *testing.T) {
	ctx := context.Background()
	w := newWrapperForTest(t)

	require.NoError(t, w.WriteScore(ctx, "//assets/1", &policy.Score{QrId: "q1", Type: policy.ScoreType_Result}))
	require.NoError(t, w.WriteScore(ctx, "//assets/1", &policy.Score{QrId: "q2", Type: policy.ScoreType_Error}))
	require.NoError(t, w.WriteData(ctx, "//assets/1", &llx.Result{CodeId: "d1"}))
	require.NoError(t, w.WriteData(ctx, "//assets/1", &llx.Result{CodeId: "d2", Error: "boom"}))

	require.Equal(t, int64(4), w.ExecutedCount()) // 2 scores + 2 data
	require.Equal(t, int64(2), w.ErroredCount())  // 1 error score + 1 error data
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./policy/scandb/ -run TestWrapper_CountsExecutedAndErrored -v`
Expected: FAIL — `ExecutedCount`/`ErroredCount` undefined.

- [ ] **Step 3: Add counters and getters to the wrapper**

In `policy/scandb/scan_data_store_wrapper.go`:

Add `"sync/atomic"` to the imports. Extend the struct:

```go
type ScanDataStoreWrapper struct {
	store    ScanDataStore
	assetMrn string

	executed atomic.Int64 // total queries written (scores + data)
	errored  atomic.Int64 // queries in an error state
}
```

In `WriteScore`, after the `validate` check and before writing, add counting:

```go
func (w *ScanDataStoreWrapper) WriteScore(ctx context.Context, assetMrn string, score *policy.Score) error {
	if err := w.validate(assetMrn); err != nil {
		return err
	}
	w.executed.Add(1)
	if score.Type == policy.ScoreType_Error {
		w.errored.Add(1)
	}
	return w.store.WriteScores(ctx, []*policy.Score{score})
}
```

In `WriteData`, likewise:

```go
func (w *ScanDataStoreWrapper) WriteData(ctx context.Context, assetMrn string, data *llx.Result) error {
	if err := w.validate(assetMrn); err != nil {
		return err
	}
	w.executed.Add(1)
	if data.GetError() != "" {
		w.errored.Add(1)
	}
	return w.store.WriteData(ctx, []*llx.Result{data})
}
```

Add getters at the end of the file:

```go
// ExecutedCount returns the number of queries (scores + data results) written.
func (w *ScanDataStoreWrapper) ExecutedCount() int64 { return w.executed.Load() }

// ErroredCount returns the number of written queries in an error state.
func (w *ScanDataStoreWrapper) ErroredCount() int64 { return w.errored.Load() }
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./policy/scandb/ -run TestWrapper_CountsExecutedAndErrored -v`
Expected: PASS.

The constructor signature is `NewSqliteScanDataStore(filePath, assetMrn string)` (`policy/scandb/scan_data_store.go:79`) — the two-arg form used here. (The wrapper's doc comment showing a third `sessionId` arg is stale; ignore it.)

- [ ] **Step 5: Commit**

```bash
git add policy/scandb/scan_data_store_wrapper.go policy/scandb/scan_data_store_wrapper_stats_test.go
git commit -m "✨ scandb: count executed and errored queries in wrapper"
```

---

## Task 4: Plumb the Collector through the upload path and attach `details`

**Files:**
- Modify: `internal/datalakes/sqlite/sqlite.go` (`WithServices`, `uploadScanDataStore`)
- Test: `internal/datalakes/sqlite/sqlite_stats_test.go` (new) — unit-test the size helper

Because `WithServices` requires a live runtime + upstream to run end to end, the wiring is verified by (a) the compile + existing tests, and (b) a focused unit test of the upload-size helper extracted below.

- [ ] **Step 1: Extract an upload-size helper and write its failing test**

Create `internal/datalakes/sqlite/sqlite_stats_test.go`:

```go
package sqlite

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy/scanstats"
)

func TestFileSizeBytes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "scan.db")
	require.NoError(t, os.WriteFile(p, []byte("hello world"), 0o600))
	require.Equal(t, int64(11), fileSizeBytes(p))
	require.Equal(t, int64(0), fileSizeBytes(filepath.Join(t.TempDir(), "missing.db")))
}

func TestCollectorUploadSize(t *testing.T) {
	c := scanstats.New()
	c.AddInt(scanstats.MetricUploadSize, "bytes", 11)
	stats := c.ToProto()
	require.Equal(t, scanstats.MetricUploadSize, stats.Metrics[0].Name)
	require.Equal(t, int64(11), stats.Metrics[0].GetIntValue())
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/datalakes/sqlite/ -run 'TestFileSizeBytes|TestCollectorUploadSize' -v`
Expected: FAIL — `fileSizeBytes` undefined.

- [ ] **Step 3: Add the helper**

In `internal/datalakes/sqlite/sqlite.go`, add near `uploadScanDataStore`:

```go
// fileSizeBytes returns the size of the file at path, or 0 if it cannot be stat'd.
func fileSizeBytes(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/datalakes/sqlite/ -run 'TestFileSizeBytes|TestCollectorUploadSize' -v`
Expected: PASS.

- [ ] **Step 5: Create and time the Collector in `WithServices`**

In `internal/datalakes/sqlite/sqlite.go`, add imports `"time"` and `"go.mondoo.com/cnspec/v13/policy/scanstats"` (keep existing imports).

In `WithServices`, keep a reference to the wrapper and create the collector. Replace the inline wrapper construction and the scan-run block. Current code:

```go
		_, ls, err := inmemory.NewServices(runtime, inmemory.WithDataWriter(scandb.NewScanDataStoreWrapper(scanDataStore, assetMrn)))
		if err != nil {
			return err
		}
```

becomes:

```go
		wrapper := scandb.NewScanDataStoreWrapper(scanDataStore, assetMrn)
		_, ls, err := inmemory.NewServices(runtime, inmemory.WithDataWriter(wrapper))
		if err != nil {
			return err
		}

		stats := scanstats.New()
```

Then wrap the `f(ctx, ls)` call with timing. Current code:

```go
		ls.Upstream = upstream
		if err := f(ctx, ls); err != nil {
			return err
		}
```

becomes:

```go
		ls.Upstream = upstream
		scanStart := time.Now()
		if err := f(ctx, ls); err != nil {
			return err
		}
		stats.AddDuration(scanstats.MetricScanDuration, time.Since(scanStart))
		stats.AddInt(scanstats.MetricQueriesExecuted, "count", wrapper.ExecutedCount())
		stats.AddInt(scanstats.MetricQueriesErrored, "count", wrapper.ErroredCount())
```

- [ ] **Step 6: Pass the collector to `uploadScanDataStore` and record upload size + attach**

Still in `WithServices`, update the upload call. Current:

```go
		if upstream != nil {
			scanDataPath, err := scanDataStore.Finalize()
			if err != nil {
				return err
			}

			return uploadScanDataStore(ctx, upstream, assetMrn, scanDataPath)
		}
```

becomes:

```go
		if upstream != nil {
			scanDataPath, err := scanDataStore.Finalize()
			if err != nil {
				return err
			}
			stats.AddInt(scanstats.MetricUploadSize, "bytes", fileSizeBytes(scanDataPath))

			return uploadScanDataStore(ctx, upstream, assetMrn, scanDataPath, stats)
		}
```

Update `uploadScanDataStore`'s signature and the `ReportUploadCompleted` call. Current signature and call:

```go
func uploadScanDataStore(ctx context.Context, services *policy.Services, assetMrn string, scanDataPath string) error {
```

becomes:

```go
func uploadScanDataStore(ctx context.Context, services *policy.Services, assetMrn string, scanDataPath string, stats *scanstats.Collector) error {
```

And the confirmation call. Current:

```go
	// Confirm the upload
	_, err = services.ReportUploadCompleted(ctx, &policy.ReportUploadCompletedReq{
		UploadSessionId: urlResp.UploadSessionId,
		ScopeMrn:        assetMrn,
	})
	if err != nil {
		return err
	}
```

becomes:

```go
	// Confirm the upload, attaching scan statistics as the completion payload.
	req := &policy.ReportUploadCompletedReq{
		UploadSessionId: urlResp.UploadSessionId,
		ScopeMrn:        assetMrn,
	}
	if s := stats.ToProto(); s != nil {
		if details, aerr := anypb.New(s); aerr != nil {
			log.Warn().Err(aerr).Msg("failed to encode scan statistics; sending upload confirmation without them")
		} else {
			req.Details = details
		}
	}
	if _, err = services.ReportUploadCompleted(ctx, req); err != nil {
		return err
	}
```

Add the import `"google.golang.org/protobuf/types/known/anypb"` to the file.

- [ ] **Step 7: Build and run the package tests**

Run: `go build ./... && go test ./internal/datalakes/sqlite/ ./policy/scandb/ ./policy/scanstats/ ./policy/ -v`
Expected: build succeeds; all tests PASS. (If `go build ./...` surfaces an unused-import or signature-mismatch error, fix per the message — every caller of `uploadScanDataStore` is inside this file.)

- [ ] **Step 8: Commit**

```bash
git add internal/datalakes/sqlite/sqlite.go internal/datalakes/sqlite/sqlite_stats_test.go
git commit -m "✨ sqlite: attach scan statistics to ReportUploadCompleted"
```

---

## Task 5: Full verification

- [ ] **Step 1: Run the full affected test suite with race detection**

Run: `go test -race ./policy/... ./internal/datalakes/sqlite/...`
Expected: PASS (race detector clean — the wrapper counters use `atomic.Int64`, the collector uses a mutex).

- [ ] **Step 2: Lint**

Run: `make test/lint` (or `golangci-lint run ./policy/... ./internal/datalakes/sqlite/...`)
Expected: no new findings.

- [ ] **Step 3: Sanity-check the wire shape (optional manual check)**

If an upstream/test environment is available, run a scan that uploads and confirm on the server side that `ReportUploadCompleted` arrives with a `details` `Any` of type URL `type.googleapis.com/cnspec.policy.v1.ScanStatistics` containing four metrics. Otherwise the Task 1 round-trip tests are the authoritative proof the payload encodes/decodes correctly.

- [ ] **Step 4: Final commit (if lint required changes)**

```bash
git add -A
git commit -m "🧹 scan statistics: lint fixes"
```

---

## Self-review notes

- **Spec coverage:** duration (Task 4 s5), queries_executed (Tasks 3+4), queries_errored (Tasks 3+4), upload_size (Task 4 s5/6); `Any details` extension point + `ScanStatistics`/`Metric` with oneof value types (Task 1); namespaced names + Collector seam (Task 2); backward compat = new optional field + nil-safe collector (Tasks 1, 2, 4); vtproto Any de-risk (Task 1 s5-7). All covered.
- **Type consistency:** `Collector` adders and `ToProto()` names match across Tasks 2 and 4; `ExecutedCount()`/`ErroredCount()` defined in Task 3 and consumed in Task 4; proto field/oneof names (`Details`, `Metric_IntValue`, `GetIntValue()`) match generated protoc-gen-go conventions used in tests.
- **Verified facts:** `NewSqliteScanDataStore(filePath, assetMrn)` two-arg signature (`scan_data_store.go:79`); vtprotobuf ships the `anypb` VT wrapper (module cache) so Task 1 codegen will compile; `mql` symlink present so `go generate ./policy` resolves proto imports.
