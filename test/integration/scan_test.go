// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
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

// allContent loads every policy and query pack in content/ with -f.
//
// The default-policy scenarios use it instead of resolving from the registry,
// so what they assert -- in particular that no check errors -- is a property of
// this checkout, not of whatever the registry serves at the time. Filters still
// decide what runs: a bundle that does not apply to the target contributes
// nothing. The upstream tier is the one place that resolves from a service, and
// it has to: cnspec switches to incognito when given -f.
//
// Scanning alpine:3.20 with all 122 bundles took 8s on a laptop (2026-09-23).
var allContent = contentBundleArgs()

func contentBundleArgs() []string {
	var args []string
	for _, g := range []string{"../../content/*.mql.yaml", "../../content/querypacks/*.mql.yaml"} {
		files, err := filepath.Glob(g)
		if err != nil {
			panic(err)
		}
		for _, f := range files {
			args = append(args, "-f", f)
		}
	}
	// A glob that silently matched nothing would turn these scenarios into
	// scans with no policies at all.
	if len(args)/2 < 100 {
		panic(fmt.Sprintf("content globs matched %d bundles; is the path still right?", len(args)/2))
	}
	return args
}

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
// check floors, no errored check) and survive a patch bump, while a minor tag keeps
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

	// timeout bounds this scenario. Zero means defaultScenarioTimeout.
	timeout time.Duration

	// assert runs only when the scan exited 0 and stdout parsed.
	assert func(t *testing.T, rep *reporter.Report)
}

// scenarios is the table. Check floors carry the count observed when they were
// set, so drift is visible without re-deriving them.
var scenarios = []scenario{
	{
		// Every policy and query pack in this repo against a pinned image. The
		// filters pick what applies to alpine, so this is the shipped content
		// the way a user on that platform meets it, and it must finish without
		// a single errored check.
		//
		// This one keeps a check floor where the local-tier scenario does not,
		// because the target is a pinned image: which checks apply is a
		// property of the content, not of the machine running the suite. If
		// this floor starts failing, the content or the filters really did
		// change.
		name: "alpine-all-content",
		tier: tierDocker, image: imageAlpine,
		args:     append([]string{"scan", "docker", imageAlpine}, allContent...),
		wantExit: 0,
		assert: func(t *testing.T, rep *reporter.Report) {
			mrn, asset := requireOneAsset(t, rep)
			requireNoAssetErrors(t, rep)
			assert.Equal(t, "alpine", asset.GetPlatformName())
			requireAssetScored(t, rep, mrn)
			requireCheckFloor(t, rep, mrn, 50) // observed 85, 2026-09-23
			requireVerdicts(t, rep, mrn, 30)   // observed 57 pass+fail
			requireNoCheckErrors(t, rep, mrn)
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
			requireNoCheckErrors(t, rep, mrn)
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
			requireNoCheckErrors(t, rep, mrn)
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
			requireNoCheckErrors(t, rep, mrn)
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
		// The local connector, against whatever the runner is, with every
		// policy and query pack in this repo.
		//
		// No check-count floor here, deliberately. Which checks apply is a
		// property of the host, not of cnspec: a developer laptop and a CI
		// runner match different groups, and a floor calibrated on either one
		// is a false failure on the other.
		//
		// What is invariant is that the local connector produced an asset, the
		// engine executed something, at least one check reached a verdict, and
		// none errored -- on whatever host this is.
		name:     "local-all-content",
		tier:     tierLocal,
		args:     append([]string{"scan", "local"}, allContent...),
		wantExit: 0, timeout: 10 * time.Minute,
		assert: func(t *testing.T, rep *reporter.Report) {
			mrn, asset := requireOneAsset(t, rep)
			requireNoAssetErrors(t, rep)
			assert.NotEmpty(t, asset.GetPlatformName())
			requireAssetScored(t, rep, mrn)
			requireVerdicts(t, rep, mrn, 1)
			requireNoCheckErrorsExcept(t, rep, mrn, nonRootForbiddenChecks())
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
			requireNoCheckErrors(t, rep, mrn)
		},
	},
}

// nonRootForbiddenChecks names the checks that error in a non-root scan of a
// Linux host because the scan may not read another user's home directory.
//
// The AI agent resources read every user's home, and /root is closed to a
// regular user: "open /root/.cursor/rules: permission denied". Under ADR-046
// (structured provider errors, mql#10973) that failure is ERROR_KIND_FORBIDDEN
// and still scores as an error, but the kind reaches the score. Once it does,
// this list goes away in favour of tolerating only forbidden errors in a
// non-root scan. Until then the checks are named here so that every other
// error still fails the local tier, which CI runs as a regular user.
//
// Measured on ubuntu:22.04 as uid 1000 with v14.0.0-rc.11 (2026-09-23).
func nonRootForbiddenChecks() map[string]string {
	if runtime.GOOS != "linux" || os.Geteuid() == 0 {
		return nil
	}
	const why = "non-root scan cannot read other users' homes (ADR-046 forbidden)"
	return map[string]string{
		"mondoo-ai-security-no-cursor":                 why,
		"mondoo-ai-security-no-goose":                  why,
		"mondoo-ai-security-no-unapproved-mcp-servers": why,
		"mondoo-ai-security-no-windsurf":               why,
		"mondoo-ai-security-no-zed":                    why,
	}
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
