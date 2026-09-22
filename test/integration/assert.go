// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reporter"
	mqlreporter "go.mondoo.com/mql/cli/reporter"
	"google.golang.org/protobuf/encoding/protojson"
)

// providerPanic is what the plugin layer logs when it recovers a panic inside a
// provider. See requireNoProviderPanic.
const providerPanic = "recovered panic in provider"

// decodeReport parses `-o json` output into the type that format actually
// emits.
//
// It is not policy.ReportCollection. `--json` and `-o json` both resolve to
// FormatJSONv2 (cli/reporter/print.go), which goes through ConvertToProto and
// protojson.Marshal (cli/reporter/proto.go), so the payload is
// cli/reporter.Report: assets, data, errors, scores. Decoding it into
// ReportCollection appears to work because the `assets` key collides, and then
// Reports, Bundle and ResolvedPolicies come back empty -- which is why an
// existing test that does so can only assert that Assets is non-empty.
//
// protojson, not encoding/json: protojson emits the JSON name while the
// generated struct tag is the proto name, so `platformName` on the wire never
// reaches the PlatformName field under encoding/json. An assertion on it would
// silently compare against "".
func decodeReport(t *testing.T, res *result) *reporter.Report {
	t.Helper()
	rep := &reporter.Report{}
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(res.stdout, rep); err != nil {
		t.Fatalf("could not decode scan report: %v\n%s", err, res.dump())
	}
	return rep
}

// requireNoProviderPanic fails when a provider panicked during the scan.
//
// The plugin layer recovers a panic in a provider method, logs it to stderr and
// answers the query with an error. The check then carries status "error", the
// asset still reports, and the process still exits 0 -- so a nil dereference in
// a resource is invisible to every other signal this suite has. This was not
// hypothetical: while building the suite, one run of mondoo-linux-security
// against ubuntu:22.04 turned 12 of 22 checks into errors from a nil pointer in
// the os provider's kernel resource, and exited 0. It is intermittent, which is
// exactly why an assertion is worth more than a manual look.
func requireNoProviderPanic(t *testing.T, res *result) {
	t.Helper()
	if n := countProviderPanics(res.stderr); n > 0 {
		t.Errorf("provider panicked %d time(s) during the scan; checks that hit it report "+
			"status \"error\" and the process still exits 0\n%s", n, res.dump())
	}
}

// countProviderPanics counts recovered provider panics in a scan's stderr.
func countProviderPanics(stderr []byte) int {
	return bytes.Count(stderr, []byte(providerPanic))
}

// requireOneAsset returns the single asset's MRN and record.
//
// The MRN cannot be predicted: an incognito scan mints a fresh identifier per
// asset, so the map's own key is the only handle.
func requireOneAsset(t *testing.T, rep *reporter.Report) (string, *mqlreporter.Asset) {
	t.Helper()
	require.Len(t, rep.GetAssets(), 1, "expected exactly one asset, got %d", len(rep.GetAssets()))
	for mrn, a := range rep.GetAssets() {
		require.NotEmpty(t, mrn, "asset has no MRN")
		return mrn, a
	}
	return "", nil
}

// requireNoAssetErrors fails when any asset failed to scan.
//
// This map is also the only thing that moves the exit code at the default risk
// threshold, so it is asserted separately from the exit code rather than
// through it: the message names the asset, the exit code does not.
func requireNoAssetErrors(t *testing.T, rep *reporter.Report) {
	t.Helper()
	if len(rep.GetErrors()) > 0 {
		t.Fatalf("scan reported asset errors: %v", rep.GetErrors())
	}
}

// checkScores returns the per-check scores for an asset, with the asset's own
// aggregate entry removed.
func checkScores(t *testing.T, rep *reporter.Report, assetMrn string) map[string]*reporter.ScoreValue {
	t.Helper()
	sv := rep.GetScores()[assetMrn]
	require.NotNil(t, sv, "no scores at all for asset %s", assetMrn)

	out := make(map[string]*reporter.ScoreValue, len(sv.GetValues()))
	for mrn, v := range sv.GetValues() {
		if mrn == assetMrn {
			continue // the asset's own aggregate score
		}
		out[mrn] = v
	}
	return out
}

// requireAssetScored asserts the asset's aggregate score exists and reached a
// verdict. A "skip" or "unscored" aggregate means nothing about the asset was
// actually evaluated.
func requireAssetScored(t *testing.T, rep *reporter.Report, assetMrn string) {
	t.Helper()
	overall := rep.GetScores()[assetMrn].GetValues()[assetMrn]
	require.NotNil(t, overall, "no aggregate score for asset %s", assetMrn)
	// riskScore is 100 - Score.Value; the proto reserved the absolute score
	// field, so this is the only number on the wire.
	assert.LessOrEqual(t, overall.GetRiskScore(), uint32(100), "risk score out of range")
	assert.Contains(t, []string{"pass", "fail"}, overall.GetStatus(),
		"asset aggregate is %q, so nothing was evaluated", overall.GetStatus())
}

// requireCheckFloor asserts at least min checks were scored.
//
// A floor, not an equality. The count moves when content is released and when a
// base image updates and a filter starts or stops matching, and neither is a
// cnspec regression. It collapses when the resolver, the executor or the
// provider breaks, and that is. Each caller records the count observed when the
// floor was set, so a drift is visible without re-deriving it.
func requireCheckFloor(t *testing.T, rep *reporter.Report, assetMrn string, min int) {
	t.Helper()
	scores := checkScores(t, rep, assetMrn)
	if len(scores) < min {
		t.Errorf("only %d checks scored, want >= %d (%s)", len(scores), min, histogram(scores))
	}
}

// requireCheckPrefixFloor asserts a specific bundle resolved, by counting the
// checks whose identifier came from it.
//
// A bundle loaded with -f in incognito gets local MRNs built from the check
// UIDs in the YAML, so the UID prefix is the stable handle. The policy's own
// score is not in this map, so "the bundle resolved" is asserted through its
// checks rather than through it.
func requireCheckPrefixFloor(t *testing.T, rep *reporter.Report, assetMrn, prefix string, min int) {
	t.Helper()
	scores := checkScores(t, rep, assetMrn)
	var n int
	for mrn := range scores {
		if strings.Contains(mrn, prefix) {
			n++
		}
	}
	if n < min {
		t.Errorf("%d checks matched %q, want >= %d: the bundle did not resolve (%s)",
			n, prefix, min, histogram(scores))
	}
}

// requireErrorRatioBelow bounds the share of checks that errored.
//
// Not zero. A check erroring on a minimal base image is normal -- a resource
// that does not exist in a container is a legitimate error, and alpine:3.20
// produces one against the default policies today. A large share means the
// provider or the executor broke, and nothing else in this suite sees it:
// errored checks do not populate the report's error map and do not move the
// exit code.
func requireErrorRatioBelow(t *testing.T, rep *reporter.Report, assetMrn string, max float64) {
	t.Helper()
	scores := checkScores(t, rep, assetMrn)
	if len(scores) == 0 {
		t.Errorf("no checks scored at all")
		return
	}
	errored := statusCounts(scores)["error"]
	ratio := float64(errored) / float64(len(scores))
	if ratio > max {
		t.Errorf("%d of %d checks errored (%.0f%%), want <= %.0f%% (%s)",
			errored, len(scores), ratio*100, max*100, histogram(scores))
	}
}

// requireVerdicts asserts at least min checks reached a real pass/fail verdict.
//
// The single most load-bearing assertion here. A scan that connects, produces
// an asset, scores every check as "skip" or "error" and exits 0 is
// indistinguishable from a healthy one by exit code, by asset count, and by
// check count -- this is what tells them apart.
func requireVerdicts(t *testing.T, rep *reporter.Report, assetMrn string, min int) {
	t.Helper()
	scores := checkScores(t, rep, assetMrn)
	verdicts := verdictCount(scores)
	if verdicts < min {
		t.Errorf("only %d checks reached a pass/fail verdict, want >= %d (%s)",
			verdicts, min, histogram(scores))
	}
}

// histogram renders the status distribution for a failure message. Statuses are
// "pass", "fail", or a score type label: "error", "skip", "unscored",
// "disabled", "out of scope", "unknown".
func histogram(scores map[string]*reporter.ScoreValue) string {
	counts := statusCounts(scores)
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, counts[k]))
	}
	return "statuses: " + strings.Join(parts, " ")
}

// statusCounts tallies check statuses. Kept separate from the assertions so the
// suite's own logic can be tested without a cluster, a daemon or a network --
// see assert_test.go. The assertions are the whole value of this suite, so they
// are worth testing directly.
func statusCounts(scores map[string]*reporter.ScoreValue) map[string]int {
	counts := make(map[string]int, len(scores))
	for _, v := range scores {
		counts[v.GetStatus()]++
	}
	return counts
}

// verdictCount is the number of checks that actually reached pass or fail.
func verdictCount(scores map[string]*reporter.ScoreValue) int {
	c := statusCounts(scores)
	return c["pass"] + c["fail"]
}
