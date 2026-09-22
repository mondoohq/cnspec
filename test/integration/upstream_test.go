// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build integration

package integration

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The upstream tier is the only part of this suite that authenticates.
//
// Everything else runs incognito, which is the right default but skips exactly
// the code this tier exists for: registering an asset, resolving the policies
// the space assigns, asking the vulnerability service, and uploading the
// report. A release that broke any of those would leave every other tier green.
//
// It exists to answer a specific question -- does a client of this version
// still work against the server version in production? The client and the
// platform are released separately and a fleet routinely runs a client that is
// ahead of the server it reports to, so "the client is newer" is the normal
// case rather than the exception, and it is not covered anywhere else.

// incognitoFallback is what cnspec logs when it finds no usable credential.
//
// This is the assertion the tier turns on. Without a credential cnspec does not
// fail -- it switches to incognito and scans happily, so every other assertion
// here would pass while testing none of the upstream path. A missing or expired
// service account has to be a failure, not a silent downgrade.
const incognitoFallback = "Switching to --incognito mode"

// upstreamAssetMrn matches an asset registered in a space. An incognito scan
// mints a local identifier instead, so the shape of the MRN is what separates
// "reported to the platform" from "scanned and kept to itself".
var upstreamAssetMrn = regexp.MustCompile(`^//assets\.[^/]+/spaces/[^/]+/assets/.+`)

// apiVersions pulls the client and server API versions out of `cnspec status`.
var (
	clientAPILine = regexp.MustCompile(`API\s+(v\d+)`)
	serverAPILine = regexp.MustCompile(`Status\s+\S+\s+API (v\d+)\s*(.*)`)
)

func requireUpstream(t *testing.T) string {
	t.Helper()
	path := strings.TrimSpace(os.Getenv(upstreamConfig))
	if path == "" {
		skipTier(t, "upstream", "no service account; set "+upstreamConfig+
			"=/path/to/serviceaccount.json to run against the platform")
	}
	if _, err := os.Stat(path); err != nil {
		// A configured-but-unusable credential is a failure, never a skip: it
		// is indistinguishable from a working one in every downstream symptom
		// except that the scan quietly goes incognito.
		t.Fatalf("%s=%q is not readable: %v", upstreamConfig, path, err)
	}
	return path
}

// TestUpstreamStatus reports the client and server API versions.
//
// Cheapest possible upstream round-trip, and run first so a credential or
// connectivity problem fails here rather than halfway through a scan. It also
// puts both versions in the CI log, which is the record of what the run
// actually proved compatible.
func TestUpstreamStatus(t *testing.T) {
	cfg := requireUpstream(t)

	res := runEnv(t, upstreamEnv(cfg), 3*time.Minute, "status")
	out := string(res.stdout) + string(res.stderr)

	if res.exitCode != 0 {
		res.saveArtifact(t, "upstream-status")
		t.Fatalf("`cnspec status` exited %d\n%s", res.exitCode, res.dump())
	}

	assert.Contains(t, out, "registered", "the service account did not register")
	assert.Contains(t, out, "SERVING", "the platform did not report itself healthy")

	if m := clientAPILine.FindStringSubmatch(out); m != nil {
		t.Logf("client API %s", m[1])
	}
	if m := serverAPILine.FindStringSubmatch(out); m != nil {
		t.Logf("server API %s %s", m[1], strings.TrimSpace(m[2]))
	}
}

// TestUpstreamScan runs a full authenticated scan and asserts it reached the
// platform rather than falling back.
func TestUpstreamScan(t *testing.T) {
	cfg := requireUpstream(t)
	requireDocker(t)
	pullImage(t, imageAlpine)

	res := runEnv(t, upstreamEnv(cfg), 15*time.Minute,
		"scan", "docker", imageAlpine,
		"--annotation", "mondoo.com/integration-suite=upstream-tier",
		"--detect-cicd=false", "-o", "json")

	defer func() {
		if t.Failed() {
			res.saveArtifact(t, "upstream-scan")
		}
	}()

	if res.exitCode != 0 {
		t.Fatalf("exit code %d, want 0\n%s", res.exitCode, res.dump())
	}
	requireNoProviderPanic(t, res)

	// Before anything else: prove this was not an incognito scan wearing an
	// upstream costume.
	require.NotContains(t, string(res.stderr), incognitoFallback,
		"cnspec fell back to incognito, so nothing upstream was exercised")

	rep := decodeReport(t, res)
	mrn, asset := requireOneAsset(t, rep)
	requireNoAssetErrors(t, rep)

	assert.Regexp(t, upstreamAssetMrn, mrn,
		"the asset carries a local identifier, so it was never registered upstream")
	assert.Equal(t, "alpine", asset.GetPlatformName())

	// Policies came from the space, so the count is the space's business and
	// not something to pin. That the engine ran them, and that they produced
	// real verdicts rather than a wave of errors, is.
	requireAssetScored(t, rep, mrn)
	requireVerdicts(t, rep, mrn, 1)
	requireErrorRatioBelow(t, rep, mrn, 0.25)

	// The upload is the last thing that can break and the first thing a broken
	// client/server pairing breaks. Asserted on stderr because the report on
	// stdout is written before the upload is attempted.
	assert.Contains(t, string(res.stderr), "uploaded scan data",
		"the report was rendered but never reached the platform")
}
