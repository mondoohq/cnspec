// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build integration

package integration

import (
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnspec/cli/reporter"
)

// Bundles are addressed relative to this package.
const (
	linuxSecurity = "../../content/mondoo-linux-security.mql.yaml"
	k8sSecurity   = "../../content/mondoo-kubernetes-security.mql.yaml"
)

// Check identifiers from a bundle loaded with -f are built from the UIDs in the
// YAML, so the UID prefix is the stable handle for "this bundle resolved".
const (
	linuxSecurityUID = "mondoo-linux-security-"
	k8sSecurityUID   = "mondoo-kubernetes-security-"
)

// Images are pinned to a minor tag, never :latest and never a digest.
//
// test/sbom/README.md records why digests and goldens were abandoned there: the
// tags are mutable, so an upstream security update broke goldens that were
// still correct about cnspec. That lesson constrains what is asserted, not how
// images are pinned -- the assertions here are structural (platform family,
// check floors, error ratio) and survive a patch bump, while a minor tag keeps
// the suite answering "does cnspec still read a current alpine", which is the
// point of a release gate.
const (
	imageAlpine = "alpine:3.20"
	imageUbuntu = "ubuntu:22.04"
	imageDebian = "debian:12-slim"
)

// defaultScenarioTimeout applies to any scenario that does not set its own.
// A container scan is minutes rather than seconds once a cold provider
// download is included; the per-scenario field is for the outliers, so the
// common value lives in one place.
const defaultScenarioTimeout = 8 * time.Minute

type tier string

const (
	tierDocker tier = "docker"
	tierLocal  tier = "local"
)

type scenario struct {
	name string
	tier tier

	// image is pulled before the scan when set.
	image string

	// args is the full argv after the binary.
	args []string

	// wantExit is asserted exactly. cnspec's default --risk-threshold is 101
	// and the comparison is `100 - worstScore >= threshold`, which 100 can
	// never satisfy -- so a scan where every check fails still exits 0. Only
	// asset errors, an explicit --risk-threshold, or a fatal produce 1.
	wantExit int

	// onlyOS restricts the scenario to these GOOS values.
	onlyOS []string

	timeout time.Duration

	// timeout bounds this scenario. Zero means defaultScenarioTimeout.

	// assert runs only when the scan exited 0 and stdout parsed.
	assert func(t *testing.T, rep *reporter.Report)
}

// scenarios is the table. Check floors carry the count observed when they were
// set, so drift is visible without re-deriving them.
var scenarios = []scenario{
	{
		// The default-policy path: no -f, so policies are resolved from the
		// production registry. Nothing else in this repo exercises that --
		// content/validation always passes an explicit bundle -- yet it is what
		// every new user hits first.
		name: "alpine-default-policies",
		tier: tierDocker, image: imageAlpine,
		args:     []string{"scan", "docker", imageAlpine},
		wantExit: 0,
		assert: func(t *testing.T, rep *reporter.Report) {
			mrn, asset := requireOneAsset(t, rep)
			requireNoAssetErrors(t, rep)
			assert.Equal(t, "alpine", asset.GetPlatformName())
			requireAssetScored(t, rep, mrn)
			requireCheckFloor(t, rep, mrn, 30) // observed 64, 2026-09-22
			requireVerdicts(t, rep, mrn, 20)   // observed 58 pass+fail
			requireErrorRatioBelow(t, rep, mrn, 0.15)
		},
	},
	{
		// The same image against a bundle from this repo. Exercises bundle load
		// -> requirement resolution -> compile -> execute through the CLI,
		// which content/validation only covers in-process.
		name: "alpine-linux-security-bundle",
		tier: tierDocker, image: imageAlpine,
		args:     []string{"scan", "docker", imageAlpine, "-f", linuxSecurity},
		wantExit: 0,
		assert: func(t *testing.T, rep *reporter.Report) {
			mrn, _ := requireOneAsset(t, rep)
			requireNoAssetErrors(t, rep)
			requireCheckPrefixFloor(t, rep, mrn, linuxSecurityUID, 10)
			requireErrorRatioBelow(t, rep, mrn, 0.20)
		},
	},
	{
		// A dpkg platform, for the other half of the package-detection code
		// path. apk and dpkg are separate provider code.
		name: "ubuntu-linux-security-bundle",
		tier: tierDocker, image: imageUbuntu,
		args:     []string{"scan", "docker", imageUbuntu, "-f", linuxSecurity},
		wantExit: 0,
		assert: func(t *testing.T, rep *reporter.Report) {
			mrn, asset := requireOneAsset(t, rep)
			requireNoAssetErrors(t, rep)
			assert.Equal(t, "ubuntu", asset.GetPlatformName())
			requireCheckPrefixFloor(t, rep, mrn, linuxSecurityUID, 15) // observed 22, 2026-09-22
			requireVerdicts(t, rep, mrn, 10)                           // observed 18 pass
			requireErrorRatioBelow(t, rep, mrn, 0.20)
		},
	},
	{
		// A minimal image: the shape where "connected but found nothing" hides.
		name: "debian-slim-linux-security-bundle",
		tier: tierDocker, image: imageDebian,
		args:     []string{"scan", "docker", imageDebian, "-f", linuxSecurity},
		wantExit: 0,
		assert: func(t *testing.T, rep *reporter.Report) {
			mrn, asset := requireOneAsset(t, rep)
			requireNoAssetErrors(t, rep)
			assert.Equal(t, "debian", asset.GetPlatformName())
			requireCheckPrefixFloor(t, rep, mrn, linuxSecurityUID, 15) // observed 22, 2026-09-22
			requireVerdicts(t, rep, mrn, 10)                           // observed 20 pass+fail
			requireErrorRatioBelow(t, rep, mrn, 0.20)
		},
	},
	{
		// The suite's own self-test. If this ever passes, the harness has
		// stopped being able to see a failure at all, and every other green
		// result in this file becomes meaningless.
		name:     "missing-image-exits-1",
		tier:     tierDocker,
		args:     []string{"scan", "docker", "ghcr.io/mondoohq/cnspec-integration-absent:v0"},
		wantExit: 1, timeout: 4 * time.Minute,
	},
	{
		// The local connector, against whatever the runner is. The floor is low
		// and the platform assertion is only non-empty because this is the one
		// scenario whose target differs between a laptop and CI.
		name:     "local-default-policies",
		tier:     tierLocal,
		args:     []string{"scan", "local"},
		wantExit: 0, timeout: 10 * time.Minute,
		assert: func(t *testing.T, rep *reporter.Report) {
			mrn, asset := requireOneAsset(t, rep)
			requireNoAssetErrors(t, rep)
			assert.NotEmpty(t, asset.GetPlatformName())
			requireAssetScored(t, rep, mrn)
			requireCheckFloor(t, rep, mrn, 20) // observed 75 on darwin, 2026-09-22
			requireVerdicts(t, rep, mrn, 10)
		},
	},
	{
		name: "local-linux-security-bundle",
		tier: tierLocal, onlyOS: []string{"linux"},
		args:     []string{"scan", "local", "-f", linuxSecurity},
		wantExit: 0, timeout: 10 * time.Minute,
		assert: func(t *testing.T, rep *reporter.Report) {
			mrn, _ := requireOneAsset(t, rep)
			requireNoAssetErrors(t, rep)
			requireCheckPrefixFloor(t, rep, mrn, linuxSecurityUID, 15)
			requireVerdicts(t, rep, mrn, 10)
			requireErrorRatioBelow(t, rep, mrn, 0.20)
		},
	},
}

func TestDockerTargets(t *testing.T) { runTier(t, tierDocker) }

func TestLocalTarget(t *testing.T) { runTier(t, tierLocal) }

// runTier executes every scenario of one tier, serially.
//
// Serially, and with no t.Parallel(). Each scenario is a separate cnspec
// process, so the in-process hazards documented in content/validation's
// TestMain do not apply; parallel processes would instead race to install the
// same provider into a shared PROVIDERS_PATH.
//
// Measured rather than assumed, on the docker tier against v14.0.0-rc.10
// (2026-09-22): serial 67s vs parallel-5 86s on a cold provider directory, and
// serial 85s vs parallel-5 111s on a warm one. Parallel is ~25-30% slower both
// ways -- a scan is CPU and container-daemon bound, not waiting on IO, so
// running five at once only adds contention, on top of duplicating the
// provider downloads a cold directory needs.
func runTier(t *testing.T, want tier) {
	if want == tierDocker {
		requireDocker(t)
	}

	for _, sc := range scenarios {
		if sc.tier != want {
			continue
		}
		t.Run(sc.name, func(t *testing.T) {
			if len(sc.onlyOS) > 0 && !slices.Contains(sc.onlyOS, runtime.GOOS) {
				t.Skipf("scenario runs on %v, this is %s", sc.onlyOS, runtime.GOOS)
			}
			if sc.image != "" {
				pullImage(t, sc.image)
			}

			// --detect-cicd=false keeps a CI runner and a laptop running the
			// same scan: otherwise CI labels and a CI/CD asset category are
			// applied in one environment and not the other.
			timeout := sc.timeout
			if timeout == 0 {
				timeout = defaultScenarioTimeout
			}

			args := append(slices.Clone(sc.args), "--detect-cicd=false", "-o", "json")
			res := run(t, timeout, args...)

			if res.exitCode != sc.wantExit {
				res.saveArtifact(t, sc.name)
				t.Fatalf("exit code %d, want %d\n%s", res.exitCode, sc.wantExit, res.dump())
			}
			requireNoProviderPanic(t, res)

			if sc.wantExit != 0 || sc.assert == nil {
				return
			}

			rep := decodeReport(t, res)
			sc.assert(t, rep)
			if t.Failed() {
				res.saveArtifact(t, sc.name)
				t.Logf("report follows:\n%s", res.dump())
			}
		})
	}
}
