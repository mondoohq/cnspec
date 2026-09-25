// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/scanwarnings"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

func TestAggregateReport(t *testing.T) {
	b := &policy.Bundle{
		Policies: []*policy.Policy{
			{
				Uid:  "policy1",
				Name: "Policy 1",
			},
		},
	}

	r := NewAggregateReporter()
	r.AddBundle(b)
	assert.Equal(t, r.bundle, b)

	b2 := &policy.Bundle{
		Policies: []*policy.Policy{
			{
				Uid:  "policy2",
				Name: "Policy 2",
			},
		},
	}

	r.AddBundle(b2)
	assert.Equal(t, r.bundle, policy.Merge(b, b2))
}

// withReportErrorFnSpy replaces the package-level reportErrorFn seam for the
// duration of a test, restoring the original afterward. reportErrorFn is
// the only way reportCriticalErrors talks to the Mondoo Platform
// (health.ReportError), so this is how a test observes what would have
// been sent without a configured service account or a live network call.
func withReportErrorFnSpy(t *testing.T, spy func(product, version, build, errMsg string, tags map[string]string)) {
	t.Helper()
	original := reportErrorFn
	reportErrorFn = spy
	t.Cleanup(func() { reportErrorFn = original })
}

// TestReportCriticalErrors_DedupesCapsAndReportsWithAssetTags is the
// regression test for what reaches health.ReportError -- and therefore the
// platform's provider.crashed record -- when an asset's provider crashes:
// one call per deduplicated message, each carrying the asset's identifying
// tags.
func TestReportCriticalErrors_DedupesCapsAndReportsWithAssetTags(t *testing.T) {
	asset := &inventory.Asset{
		Mrn:         "//assets/1",
		Name:        "flaky-host",
		PlatformIds: []string{"platform-id-1"},
		Platform:    &inventory.Platform{Name: "windows", Version: "10.0.19045"},
	}

	type call struct {
		msg  string
		tags map[string]string
	}
	var calls []call
	withReportErrorFnSpy(t, func(product, version, build, errMsg string, tags map[string]string) {
		calls = append(calls, call{msg: errMsg, tags: tags})
	})

	errs := []error{
		errors.New("the 'os' provider crashed: connection refused"),
		errors.New("the 'os' provider crashed: connection refused"),
		errors.New("the 'os' provider crashed: connection refused"),
		errors.New("the 'aws' provider crashed: EOF"),
	}

	reportCriticalErrors(asset, errs)

	require.Len(t, calls, 2, "3 repeats of one crash + 1 distinct crash = 2 deduplicated reports")
	msgs := []string{calls[0].msg, calls[1].msg}
	assert.Contains(t, msgs, "the 'os' provider crashed: connection refused")
	assert.Contains(t, msgs, "the 'aws' provider crashed: EOF")

	for _, c := range calls {
		assert.Equal(t, asset.Mrn, c.tags["assetMrn"])
		assert.Equal(t, asset.Name, c.tags["assetName"])
		assert.Equal(t, "platform-id-1", c.tags["platformIDs"])
		assert.Equal(t, "windows", c.tags["assetPlatform"])
		assert.Equal(t, "10.0.19045", c.tags["assetPlatformVersion"])
	}
}

func TestReportCriticalErrors_NoErrorsIsNoOp(t *testing.T) {
	asset := &inventory.Asset{Mrn: "//assets/1", Name: "healthy-host"}

	var called bool
	withReportErrorFnSpy(t, func(product, version, build, errMsg string, tags map[string]string) {
		called = true
	})

	reportCriticalErrors(asset, nil)
	assert.False(t, called, "no critical errors must mean no report to the platform")
}

// TestReportCriticalErrors_CapsCount is the regression test for this call
// site applying scanwarnings.Max: a crash storm on one asset must send at
// most Max reports to the platform, not one per underlying error.
func TestReportCriticalErrors_CapsCount(t *testing.T) {
	asset := &inventory.Asset{Mrn: "//assets/1", Name: "crash-storm-host"}

	var calls int
	withReportErrorFnSpy(t, func(product, version, build, errMsg string, tags map[string]string) {
		calls++
	})

	errs := make([]error, 0, scanwarnings.Max+10)
	for i := 0; i < scanwarnings.Max+10; i++ {
		errs = append(errs, fmt.Errorf("distinct crash #%d", i))
	}

	reportCriticalErrors(asset, errs)
	assert.Equal(t, scanwarnings.Max, calls)
}

// TestReportCriticalErrors_CapsMessageLength is the regression test for
// this call site applying scanwarnings.MaxLen to each message before it
// reaches the platform.
func TestReportCriticalErrors_CapsMessageLength(t *testing.T) {
	asset := &inventory.Asset{Mrn: "//assets/1", Name: "verbose-crash-host"}

	var captured string
	withReportErrorFnSpy(t, func(product, version, build, errMsg string, tags map[string]string) {
		captured = errMsg
	})

	long := strings.Repeat("x", scanwarnings.MaxLen+500)
	reportCriticalErrors(asset, []error{errors.New(long)})

	assert.Len(t, captured, scanwarnings.MaxLen)
}

// TestReportCriticalErrors_DoesNotAffectReporterOrExitCode guards against a
// future regression that wires a crashed provider back into AddScanError (or
// any other Reporter call): reportCriticalErrors takes no Reporter at all,
// so a reporter that already has a successful report for this asset must be
// completely unaffected by it running -- Ok stays true, Errors stays empty,
// and the report survives. apps/cnspec/cmd/scan.go's exit code comes
// straight from Reports().Result.Full.Errors, so this is what keeps a crash
// warning from flipping a successful scan's exit code.
func TestReportCriticalErrors_DoesNotAffectReporterOrExitCode(t *testing.T) {
	asset := &inventory.Asset{Mrn: "//assets/1", Name: "crashed-but-scored"}

	r := NewAggregateReporter()
	r.AddReport(asset, &AssetReport{
		Mrn:    asset.Mrn,
		Report: &policy.Report{Score: &policy.Score{Value: 80}},
	})

	withReportErrorFnSpy(t, func(product, version, build, errMsg string, tags map[string]string) {})

	reportCriticalErrors(asset, []error{errors.New("the 'os' provider crashed: connection refused")})

	result := r.Reports()
	require.True(t, result.Ok, "a crash warning must not flip Reports().Ok / the CLI exit code")

	full := result.GetFull()
	require.NotNil(t, full)
	assert.Empty(t, full.Errors, "a crash warning must never appear in the errors map that drives the exit code")
	assert.Contains(t, full.Reports, asset.Mrn, "the report must survive the crash, not be dropped")
	assert.Equal(t, uint32(80), full.Reports[asset.Mrn].Score.Value)
}

func TestAggregateReportKeepsAssetErrorKinds(t *testing.T) {
	r := NewAggregateReporter()
	throttled := &inventory.Asset{Mrn: "//assets/throttled", Name: "throttled"}
	broken := &inventory.Asset{Mrn: "//assets/broken", Name: "broken"}
	r.AddScanError(throttled, fmt.Errorf("connecting: %w", llx.TooManyRequests(errors.New("slow down"),
		llx.WithScope(llx.ErrorScope_ERROR_SCOPE_ASSET, "index.docker.io"))))
	r.AddScanError(broken, errors.New("boom"))

	full := r.Reports().GetFull()
	require.NotNil(t, full)
	// Every message stays where it was.
	assert.Equal(t, "connecting: slow down", full.Errors[throttled.Mrn])
	assert.Equal(t, "boom", full.Errors[broken.Mrn])
	// Only the classified one has a detail.
	require.Len(t, full.ErrorDetails, 1)
	assert.Equal(t, llx.ErrorKind_ERROR_KIND_TOO_MANY_REQUESTS, full.ErrorDetails[throttled.Mrn].Kind)
	assert.Equal(t, "index.docker.io", full.ErrorDetails[throttled.Mrn].ScopeId)
}

func TestAggregateReportWithoutClassifiedErrorsHasNoDetails(t *testing.T) {
	r := NewAggregateReporter()
	r.AddScanError(&inventory.Asset{Mrn: "//assets/broken"}, errors.New("boom"))
	assert.Nil(t, r.Reports().GetFull().ErrorDetails)
}

func TestPullsFromRegistry(t *testing.T) {
	for _, typ := range []string{"docker-image", "docker-registry", "container-registry", "registry-image"} {
		assert.True(t, pullsFromRegistry(&inventory.Asset{Connections: []*inventory.Config{{Type: typ}}}), typ)
	}
	assert.False(t, pullsFromRegistry(&inventory.Asset{Connections: []*inventory.Config{{Type: "aws"}}}))
	assert.False(t, pullsFromRegistry(&inventory.Asset{}))
}
