# Scan statistics on `ReportUploadCompleted` — design

Date: 2026-07-17
Status: approved design, pending spec review

## Goal

Send per-scan statistics to the Mondoo platform as part of the
`ReportUploadCompleted` RPC that cnspec already makes at the end of a scan
upload. For v1 we want three metrics:

- how long the scan took,
- how many queries were executed,
- how big the upload is (the scan database file size).

The mechanism must make it **easy to add more metrics later** — including
metrics that, in the future, are contributed by providers (e.g. number of
cloud API calls made, how many times the provider was throttled, peak memory
usage). Adding a new metric should not require changing the base RPC message,
and provider-contributed metrics must be namespaced so they can never collide
with core cnspec metrics.

## Constraints discovered

- `ReportUploadCompletedReq` lives in `policy/cnspec_policy.proto` (cnspec-owned,
  `syntax = "proto3"`), currently:
  ```protobuf
  message ReportUploadCompletedReq {
    string upload_session_id = 1;
    string scope_mrn = 2;
  }
  ```
- The call is **generic across upload kinds**, not scan-specific. `GetUploadURL`
  takes an `UploadURLKind` enum with `SCAN_DATABASE_V0`, `SERVERLESS_LOGS_V0`,
  and `THIRD_PARTY_FINDINGS`. Scan is only one kind, so scan statistics must
  **not** be hardcoded as fields on the base message.
- The only Go call site is `uploadScanDataStore(...)` in
  `internal/datalakes/sqlite/sqlite.go`, invoked from `WithServices(...)` after
  the SQLite scan database is finalized to a path. This path runs **only when
  `upstream != nil`** (uploading to the platform); incognito/local scans do not
  upload, so statistics collection is naturally scoped to the upload case.
- RPC bodies are marshaled with **stock protobuf** by ranger
  (`pb.Unmarshal` + `protojson` fallback in `policy/scan/scan.ranger.go`), not
  field-by-field codegen. So `oneof` and well-known types round-trip normally.
  `ScanResult` already ships a `oneof` over a ranger RPC in production.
- There is **no existing use** of `google.protobuf.Any`, proto2, or `extend`
  anywhere in cnspec or the mql/cnquery protos. The `Any` approach below is a
  new pattern for this repo — see the de-risking task.

## Design

### Level 1 — extensible completion payload (base message stays generic)

Attach an open-ended, per-upload-kind payload to the base RPC via
`google.protobuf.Any`. The base message never changes when a new upload kind
adds its own completion payload.

```protobuf
import "google/protobuf/any.proto";

message ReportUploadCompletedReq {
  string upload_session_id = 1;
  string scope_mrn = 2;
  google.protobuf.Any details = 3;   // packs a ScanStatistics for scan uploads
}
```

The server type-switches on the `Any` type URL. For a scan-database upload the
`details` field carries a `ScanStatistics`. A future serverless-logs or
third-party-findings upload can pack a different message with no change to
`ReportUploadCompletedReq`.

### Level 2 — scan statistics as a flat, namespaced metric list

No typed core fields. Every metric — including the three v1 ones — is a named,
namespaced entry, so adding a metric never touches the proto.

```protobuf
message ScanStatistics {
  repeated Metric metrics = 1;
}

message Metric {
  string name = 1;   // namespaced: "cnspec.scan.queries_executed", "provider.aws.api_calls"
  string unit = 2;   // optional hint: "ms", "bytes", "count", ...
  oneof value {
    int64  int_value    = 3;
    double double_value = 4;
    bool   bool_value   = 5;
    string string_value = 6;
  }
}
```

**Value types.** The `oneof` covers the cases we can foresee:

- `int_value` — counts, durations (ms), byte sizes, unix timestamps.
- `double_value` — ratios/averages (cache hit rate, average query latency).
- `bool_value` — flags (e.g. "hit memory ceiling").
- `string_value` — categorical values (region, provider version). Kept for
  completeness but discouraged; these blur into metadata rather than metrics.

**Namespacing convention.** Core cnspec metrics use the `cnspec.scan.*`
namespace; provider-contributed metrics use `provider.<name>.*`. Well-known
names are Go constants (below) so the server matches on shared identifiers
rather than magic strings.

### Collection — a `Collector` plumbed through the upload path

A small, concurrency-safe collector with convenience adders. This is the seam
that makes new metrics easy — core code and (later) providers just call an
adder with a namespaced name.

```go
// new package, e.g. policy/scanstats
type Collector struct { /* mutex-guarded list of metrics */ }

func New() *Collector

func (c *Collector) AddInt(name, unit string, v int64)
func (c *Collector) AddDuration(name string, d time.Duration) // int_value in ms, unit "ms"
func (c *Collector) AddDouble(name, unit string, v float64)
func (c *Collector) AddBool(name string, v bool)

func (c *Collector) ToProto() *policy.ScanStatistics          // nil-safe: nil -> nil
```

Well-known metric-name constants (same package):

```go
const (
    MetricScanDuration     = "cnspec.scan.duration"          // ms
    MetricQueriesExecuted  = "cnspec.scan.queries_executed"  // count
    MetricUploadSize       = "cnspec.scan.upload_size"       // bytes
)
```

### Data flow

1. `WithServices` creates a `Collector`.
2. It times the scan closure `f(ctx, ls)` and records
   `AddDuration(MetricScanDuration, elapsed)`.
3. After the scan, it records the query count with
   `AddInt(MetricQueriesExecuted, "count", n)`. Source: the count of
   scores + data results written to the scan data store (fallback: resolved
   policy `QueryCounts`). Exact source finalized during implementation.
4. It hands the `Collector` to `uploadScanDataStore`.
5. `uploadScanDataStore` stats the finalized file and records
   `AddInt(MetricUploadSize, "bytes", os.Stat(scanDataPath).Size())` — there is
   already a `DEBUG_PROVIDER_MEMORY` stat call at this spot to reuse the
   pattern.
6. It builds the payload: `stats := collector.ToProto()`; if non-nil,
   `details, _ := anypb.New(stats)` and sets it on the request:
   ```go
   services.ReportUploadCompleted(ctx, &policy.ReportUploadCompletedReq{
       UploadSessionId: urlResp.UploadSessionId,
       ScopeMrn:        assetMrn,
       Details:         details,
   })
   ```

### Backward compatibility

`details` is a new optional field (#3). Servers that do not read it are
unaffected. When the collector produces no metrics (`ToProto()` returns nil),
`details` is left unset and the call is byte-for-byte identical to today.

### Future extension (not implemented in v1)

Provider-contributed metrics (API call counts, throttle counts, peak memory)
will flow through the same `Collector`: after a scan, cnspec reads counters the
provider exposes and calls `AddInt("provider.<name>.api_calls", "count", n)`
etc. No proto change is required — only the wiring that reads provider counters.
This design deliberately stops at defining the seam; the provider-side counter
mechanism is out of scope here.

## Testing

- Unit-test `Collector` → `ScanStatistics`: each adder produces the expected
  `Metric` (name, unit, correct `oneof` arm); `AddDuration` converts to ms;
  empty collector → nil proto.
- **De-risk (do first):** round-trip a `ReportUploadCompletedReq` with a
  populated `Any(ScanStatistics)` through the generated marshal/unmarshal
  (proto + vtproto if a vtproto file is generated for this message), asserting
  the `ScanStatistics` unpacks equal. This validates that the new `Any` pattern
  works end-to-end with cnspec's codegen toolchain before building on it.
- Wiring assertion: after `Finalize()`, `uploadScanDataStore` sets
  `cnspec.scan.upload_size` to the on-disk file size.

## Open implementation questions

- Exact source for `queries_executed` (scan-db row count vs resolved-policy
  `QueryCounts`) — decide during implementation; either lands as the same
  metric name.
- Whether `google.protobuf.Any` needs any addition to the `make cnspec/generate`
  proto include paths — verified as part of the de-risk task.
