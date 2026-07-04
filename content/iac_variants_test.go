// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build iac_variants

// This IaC-variant fixture suite is intentionally isolated from the main cnspec
// app test build behind the `iac_variants` build tag: it downloads extra
// providers (bicep) and runs hundreds of provider-backed scans, so it must run
// only via its dedicated make target / GitHub Actions workflow, never as part of
// the default `go test ./...`.
package content

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/scan"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// init registers the providers only this suite needs, so the default test build
// (main cnspec app test) never downloads them. See extraProviders in
// bundles_test.go. bicep backs the Bicep variants; os backs the Dockerfile
// suite (the docker-file connection lives in the os provider).
func init() {
	extraProviders = append(extraProviders, "bicep", "os")
}

// tfVariantsRoot holds one directory per IaC variant, named after the
// variant's query uid, each with a pass/ and/or fail/ fixture. See
// docs on the IaC-variant test harness for the authoring workflow.
const tfVariantsRoot = "./iac-variant-testdata"

// queryMrnPrefix is how a variant uid maps to its score MRN in report.Scores.
// Confirmed empirically: //policy.api.mondoo.app/queries/<variant-uid>.
const queryMrnPrefix = "//policy.api.mondoo.app/queries/"

// tfVariantPolicy binds a variant-uid prefix to the bundle that defines it.
type tfVariantPolicy struct {
	slugPrefix string
	bundleFile string
	policyMrn  string
}

var tfVariantPolicies = []tfVariantPolicy{
	{"mondoo-aws-security-", "./mondoo-aws-security.mql.yaml", "//policy.api.mondoo.app/policies/mondoo-aws-security"},
	{"mondoo-gcp-security-", "./mondoo-gcp-security.mql.yaml", "//policy.api.mondoo.app/policies/mondoo-gcp-security"},
	{"mondoo-azure-security-", "./mondoo-azure-security.mql.yaml", "//policy.api.mondoo.app/policies/mondoo-azure-security"},
	// Other policies with terraform variants. Empty policyMrn scans the single
	// policy in each bundle without a fragile MRN filter.
	{"mondoo-oci-security-", "./mondoo-oci-security.mql.yaml", ""},
	{"mondoo-vmware-vsphere-", "./mondoo-vmware-vsphere.mql.yaml", ""},
	{"mondoo-vmware-vsphere-esxi-", "./mondoo-vmware-vsphere-esxi.mql.yaml", ""},
	{"mondoo-okta-security-", "./mondoo-okta-security.mql.yaml", ""},
	{"mondoo-openstack-security-", "./mondoo-openstack-security.mql.yaml", ""},
	{"mondoo-gitlab-security-", "./mondoo-gitlab-security.mql.yaml", ""},
	{"mondoo-cloudflare-security-", "./mondoo-cloudflare-security.mql.yaml", ""},
	{"mondoo-github-security-", "./mondoo-github-security.mql.yaml", ""},
	{"mondoo-digitalocean-security-", "./mondoo-digitalocean-security.mql.yaml", ""},
	{"mondoo-unifi-security-", "./mondoo-unifi-security.mql.yaml", ""},
	{"mondoo-portainer-security-", "./mondoo-portainer-security.mql.yaml", ""},
	{"mondoo-snowflake-security-", "./mondoo-snowflake-security.mql.yaml", ""},
	{"mondoo-hetzner-security-", "./mondoo-hetzner-security.mql.yaml", ""},
	{"mondoo-tailscale-security-", "./mondoo-tailscale-security.mql.yaml", ""},
	{"mondoo-m365-security-", "./mondoo-m365-security.mql.yaml", ""},
}

// checkOutcome distinguishes a check that ran and passed, ran and failed, or was
// skipped (not applicable to the fixture). Skipped is treated as a fixture bug:
// the fixture must contain a resource matched by the variant's filter.
type checkOutcome int

const (
	outcomeSkipped checkOutcome = iota
	outcomePassed
	outcomeFailed
)

func (o checkOutcome) String() string {
	switch o {
	case outcomePassed:
		return "passed"
	case outcomeFailed:
		return "failed"
	default:
		return "skipped"
	}
}

// checkResult reads a single check's outcome from a scan report. Score semantics
// match TestBundles: a check that actually ran has ScoreCompletion==100 and
// Weight>0; Value==100 is a pass, anything less is a failure.
func checkResult(report *policy.Report, mrn string) (checkOutcome, uint32) {
	s := report.Scores[mrn]
	if s == nil || s.ScoreCompletion == 0 || s.Weight == 0 {
		return outcomeSkipped, 0
	}
	if s.Value == 100 {
		return outcomePassed, s.Value
	}
	return outcomeFailed, s.Value
}

// tfAssetForVariant builds a scan asset for a variant fixture, picking the
// connection type from the variant-uid suffix. terraform-hcl fixtures are a
// directory of .tf files; cloudformation fixtures point at a single template
// file (.yaml/.yml/.json) in the scenario dir.
func tfAssetForVariant(uid, path string) *inventory.Asset {
	connType := "terraform-hcl"
	switch {
	case strings.HasSuffix(uid, "-terraform-plan"):
		connType = "terraform-plan"
	case strings.HasSuffix(uid, "-terraform-state"):
		connType = "terraform-state"
	case strings.HasSuffix(uid, "-cloudformation"):
		connType = "cloudformation"
		path = cfnTemplatePath(path)
	case strings.HasSuffix(uid, "-bicep"):
		connType = "bicep"
		path = bicepTemplatePath(path)
	}
	return &inventory.Asset{
		Connections: []*inventory.Config{{
			Type:    connType,
			Options: map[string]string{"path": path},
		}},
	}
}

// cfnTemplatePath returns the single CloudFormation template file inside a
// scenario dir. The cloudformation provider takes a template PATH, not a dir.
func cfnTemplatePath(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return dir
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch {
		case strings.HasSuffix(e.Name(), ".yaml"), strings.HasSuffix(e.Name(), ".yml"),
			strings.HasSuffix(e.Name(), ".json"), strings.HasSuffix(e.Name(), ".template"):
			return filepath.Join(dir, e.Name())
		}
	}
	return dir
}

// bicepTemplatePath returns the single Bicep template file inside a scenario
// dir. The bicep provider takes a .bicep file PATH, not a dir.
func bicepTemplatePath(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return dir
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".bicep") {
			return filepath.Join(dir, e.Name())
		}
	}
	return dir
}

// scenario is a single fixture directory to scan, with its expected outcome.
type scenario struct {
	name string       // subtest name, e.g. "pass/aes256"
	dir  string       // directory of .tf files to scan
	want checkOutcome // outcomePassed for pass scenarios, outcomeFailed for fail
}

// scenariosFor returns every pass and fail scenario for a variant, located under
// <tfVariantsRoot>/<policyDir>/<uid>/{pass,fail}/. A variant may have N pass and N
// fail scenarios: each immediate subdirectory containing fixture files is one
// scenario. As a convenience, fixture files placed directly in pass/ or fail/
// count as a single scenario named "pass" (or "fail").
func scenariosFor(policyDir, uid string) []scenario {
	var out []scenario
	for _, kind := range []struct {
		name string
		want checkOutcome
	}{
		{"pass", outcomePassed},
		{"fail", outcomeFailed},
	} {
		base := filepath.Join(tfVariantsRoot, policyDir, uid, kind.name)
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}

		var subScenarios []scenario
		baseHasFixture := false
		for _, e := range entries {
			if e.IsDir() {
				dir := filepath.Join(base, e.Name())
				if dirHasFixture(dir) {
					subScenarios = append(subScenarios, scenario{
						name: kind.name + "/" + e.Name(),
						dir:  dir,
						want: kind.want,
					})
				}
			} else if isFixtureFile(e.Name()) {
				baseHasFixture = true
			}
		}

		switch {
		case len(subScenarios) > 0:
			out = append(out, subScenarios...)
		case baseHasFixture:
			out = append(out, scenario{name: kind.name, dir: base, want: kind.want})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// The suites below validate policy checks against per-check pass/fail fixtures
// under tfVariantsRoot. Each IaC type has its own runnable test name so it can
// be run in isolation (and as a separate CI matrix job):
//
//	go test -tags iac_variants ./content -run TestTerraformVariants
//	go test -tags iac_variants ./content -run TestCloudFormationVariants
//	go test -tags iac_variants ./content -run TestBicepVariants
//	go test -tags iac_variants ./content -run TestDockerfileVariants
//	go test -tags iac_variants ./content -run TestKubernetesManifestVariants

// TestTerraformVariants validates every Terraform variant (terraform-hcl,
// terraform-plan, terraform-state) fixture.
func TestTerraformVariants(t *testing.T) {
	runVariantSuite(t, "terraform", []string{"-terraform-hcl", "-terraform-plan", "-terraform-state"})
}

// TestCloudFormationVariants validates every CloudFormation variant fixture.
func TestCloudFormationVariants(t *testing.T) {
	runVariantSuite(t, "cloudformation", []string{"-cloudformation"})
}

// TestBicepVariants validates every Bicep variant fixture.
func TestBicepVariants(t *testing.T) {
	runVariantSuite(t, "bicep", []string{"-bicep"})
}

// TestDockerfileVariants validates the Dockerfile policies against per-check
// Dockerfile fixtures. Unlike the cloud variants, the Dockerfile policies are
// native single-platform (no variant suffix), so every check uid with a fixture
// dir is exercised.
func TestDockerfileVariants(t *testing.T) {
	runFixtureSuite(t, "dockerfile", dockerfilePolicies, matchAll, dockerfileAsset)
}

// TestKubernetesManifestVariants validates the Kubernetes policy against
// per-check manifest fixtures. Scanning a manifest discovers one asset per
// resource it declares (k8s-pod, k8s-deployment, …), so the check outcome is
// resolved across every discovered asset's report.
func TestKubernetesManifestVariants(t *testing.T) {
	runFixtureSuite(t, "k8s-manifest", kubernetesPolicies, matchAll, k8sManifestAsset)
}

// runVariantSuite runs the fixture suite for every variant whose uid ends in one
// of the given suffixes, across all cloud-variant policies.
func runVariantSuite(t *testing.T, label string, suffixes []string) {
	runFixtureSuite(t, label, tfVariantPolicies, func(uid string) bool {
		return uidMatchesAny(uid, suffixes)
	}, tfAssetForVariant)
}

// runFixtureSuite is the shared engine behind every IaC suite. For each policy
// it discovers uid subdirectories under tfVariantsRoot that satisfy uidMatch,
// then for each pass/fail scenario builds a scan asset with assetFor, scans it,
// and asserts the check produced the expected outcome (pass scenarios score 100,
// fail scenarios score < 100). A skipped check (no asset matched the check's
// filter) is treated as a fixture bug.
func runFixtureSuite(
	t *testing.T,
	label string,
	policies []tfVariantPolicy,
	uidMatch func(uid string) bool,
	assetFor func(uid, dir string) *inventory.Asset,
) {
	ran := false
	for _, pol := range policies {
		policyDir := strings.TrimSuffix(pol.slugPrefix, "-")
		entries, err := os.ReadDir(filepath.Join(tfVariantsRoot, policyDir))
		if err != nil {
			continue // no fixtures for this policy yet
		}

		var uids []string
		for _, e := range entries {
			if e.IsDir() && uidMatch(e.Name()) {
				uids = append(uids, e.Name())
			}
		}
		sort.Strings(uids)

		for _, uid := range uids {
			mrn := queryMrnPrefix + uid
			for _, sc := range scenariosFor(policyDir, uid) {
				ran = true
				t.Run(policyDir+"/"+uid+"/"+sc.name, func(t *testing.T) {
					t.Parallel()
					reports, err := runBundleReports(pol.bundleFile, pol.policyMrn, assetFor(uid, sc.dir))
					require.NoError(t, err)

					outcome, value := checkResultAcross(reports, mrn)
					if outcome == outcomeSkipped {
						t.Fatalf("check did not run against %s\n"+
							"the fixture likely does not contain a resource matched by the check's filter\n"+
							"check: %s", sc.dir, uid)
					}
					if outcome != sc.want {
						t.Fatalf("expected %s but got %s (score=%d)\nfixture: %s\ncheck: %s\ndefined in: %s",
							sc.want, outcome, value, sc.dir, uid, pol.bundleFile)
					}
				})
			}
		}
	}
	if !ran {
		t.Skipf("no %s fixtures found under %s", label, tfVariantsRoot)
	}
}

// uidMatchesAny reports whether uid ends in one of the given suffixes.
func uidMatchesAny(uid string, suffixes []string) bool {
	for _, s := range suffixes {
		if strings.HasSuffix(uid, s) {
			return true
		}
	}
	return false
}

// matchAll accepts every uid; used by native single-platform suites (Dockerfile,
// Kubernetes) whose checks carry no variant suffix.
func matchAll(string) bool { return true }

// dockerfilePolicies are the native Dockerfile policies, tested per check uid.
var dockerfilePolicies = []tfVariantPolicy{
	{"mondoo-dockerfile-security-", "./mondoo-dockerfile-security.mql.yaml", ""},
	{"mondoo-dockerfile-best-practices-", "./mondoo-dockerfile-best-practices.mql.yaml", ""},
}

// kubernetesPolicies is the native Kubernetes policy, tested per check uid
// against manifest fixtures.
var kubernetesPolicies = []tfVariantPolicy{
	{"mondoo-kubernetes-security-", "./mondoo-kubernetes-security.mql.yaml", "//policy.api.mondoo.app/policies/mondoo-kubernetes-security"},
}

// dockerfileAsset builds a scan asset for a Dockerfile fixture. The docker-file
// connection (os provider) takes the path to a Dockerfile.
func dockerfileAsset(_ /*uid*/, dir string) *inventory.Asset {
	// The docker-file connection reads the target from Config.Path, not Options.
	return &inventory.Asset{
		Connections: []*inventory.Config{{
			Type: "docker-file",
			Path: dockerfilePath(dir),
		}},
	}
}

// k8sManifestAsset builds a scan asset for a Kubernetes manifest fixture. The
// k8s provider discovers one asset per resource declared in the manifest.
func k8sManifestAsset(_ /*uid*/, dir string) *inventory.Asset {
	return &inventory.Asset{
		Connections: []*inventory.Config{{
			Type:     "k8s",
			Options:  map[string]string{"path": manifestPath(dir)},
			Discover: &inventory.Discovery{Targets: []string{"auto"}},
		}},
	}
}

// dockerfilePath returns the Dockerfile inside a scenario dir. The docker-file
// connection wants the file path, not the directory.
func dockerfilePath(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return dir
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "Dockerfile") {
			return filepath.Join(dir, e.Name())
		}
	}
	return dir
}

// manifestPath returns the single manifest file inside a scenario dir, or the
// dir itself if none is found (the k8s provider accepts either).
func manifestPath(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return dir
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml") {
			return filepath.Join(dir, e.Name())
		}
	}
	return dir
}

// runBundleReports scans an asset against a bundle and returns every asset
// report the scan produced. Single-asset connections (terraform, cloudformation,
// bicep, docker-file) yield one report; a Kubernetes manifest yields one per
// discovered resource. checkResultAcross then locates the report that actually
// ran the check.
func runBundleReports(policyBundlePath, policyMrn string, asset *inventory.Asset) ([]*policy.Report, error) {
	ctx := context.Background()
	policyBundle, err := policy.DefaultBundleLoader().BundleFromPaths(policyBundlePath)
	if err != nil {
		return nil, err
	}
	policyBundle.OwnerMrn = "//policy.api.mondoo.app"

	policyFilters := []string{}
	if policyMrn != "" {
		policyFilters = append(policyFilters, policyMrn)
	}

	scanner := scan.NewLocalScanner()
	result, err := scanner.RunIncognito(ctx, &scan.Job{
		Inventory: &inventory.Inventory{
			Spec: &inventory.InventorySpec{Assets: []*inventory.Asset{asset}},
		},
		Bundle:        policyBundle,
		PolicyFilters: policyFilters,
		ReportType:    scan.ReportType_FULL,
	})
	if err != nil {
		return nil, err
	}

	full := result.GetFull()
	if full == nil {
		return nil, errors.New("no full report generated")
	}
	if len(full.Errors) > 0 {
		msg := ""
		for _, e := range full.Errors {
			msg += e + "; "
		}
		return nil, errors.New("errors during scan: " + msg)
	}

	reports := make([]*policy.Report, 0, len(full.Reports))
	for _, r := range full.Reports {
		reports = append(reports, r)
	}
	if len(reports) == 0 {
		return nil, errors.New("no report found")
	}
	return reports, nil
}

// checkResultAcross returns the check outcome from whichever report actually ran
// it (a manifest scan produces several reports, one per discovered resource).
func checkResultAcross(reports []*policy.Report, mrn string) (checkOutcome, uint32) {
	for _, r := range reports {
		if outcome, value := checkResult(r, mrn); outcome != outcomeSkipped {
			return outcome, value
		}
	}
	return outcomeSkipped, 0
}

// TestTerraformVariantCoverage reports how many -terraform-hcl variants in each
// registered policy have pass+fail fixtures. It never fails the build; it exists
// to make coverage visible as the suite grows.
// iacVariantSuffixes are the infrastructure-as-code variant kinds the harness
// validates via fixture files.
var iacVariantSuffixes = []string{"-terraform-hcl", "-cloudformation", "-bicep"}

func iacSuffix(uid string) (string, bool) {
	for _, s := range iacVariantSuffixes {
		if strings.HasSuffix(uid, s) {
			return s, true
		}
	}
	return "", false
}

func TestTerraformVariantCoverage(t *testing.T) {
	for _, pol := range tfVariantPolicies {
		policyDir := strings.TrimSuffix(pol.slugPrefix, "-")
		bundle, err := policy.DefaultBundleLoader().BundleFromPaths(pol.bundleFile)
		require.NoError(t, err)

		// per-suffix tally: total and covered
		total := map[string]int{}
		covered := map[string]int{}
		missing := map[string][]string{}
		for _, q := range bundle.Queries {
			suffix, ok := iacSuffix(q.Uid)
			if !ok {
				continue
			}
			total[suffix]++
			var hasPass, hasFail bool
			for _, sc := range scenariosFor(policyDir, q.Uid) {
				if sc.want == outcomePassed {
					hasPass = true
				} else {
					hasFail = true
				}
			}
			// Some checks assert exactly what their filter requires, so no failing
			// input exists. Such variants carry a fail/IMPOSSIBLE.md marker and are
			// considered fail-covered.
			if !hasFail && failIsImpossible(policyDir, q.Uid) {
				hasFail = true
			}
			if hasPass && hasFail {
				covered[suffix]++
			} else {
				missing[suffix] = append(missing[suffix], q.Uid)
			}
		}
		for _, suffix := range iacVariantSuffixes {
			if total[suffix] == 0 {
				continue
			}
			pct := float64(covered[suffix]) / float64(total[suffix]) * 100
			t.Logf("%s %s: %d/%d covered (%.1f%%)", policyDir, suffix, covered[suffix], total[suffix], pct)
			if len(missing[suffix]) > 0 && testing.Verbose() {
				sort.Strings(missing[suffix])
				t.Logf("  uncovered (%d):\n%s", len(missing[suffix]), strings.Join(indent(missing[suffix]), "\n"))
			}
		}
	}
}

// failIsImpossible reports whether a variant is marked as having no possible
// failing input (a fail/IMPOSSIBLE.md marker exists).
func failIsImpossible(policyDir, uid string) bool {
	_, err := os.Stat(filepath.Join(tfVariantsRoot, policyDir, uid, "fail", "IMPOSSIBLE.md"))
	return err == nil
}

// isFixtureFile reports whether name is an IaC fixture file the harness scans:
// a Terraform .tf, a CloudFormation/Kubernetes template (.yaml/.yml/.json/
// .template), a Bicep .bicep, or a Dockerfile.
func isFixtureFile(name string) bool {
	if strings.HasPrefix(name, "Dockerfile") {
		return true
	}
	for _, ext := range []string{".tf", ".yaml", ".yml", ".json", ".template", ".bicep"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// dirHasFixture reports whether dir contains at least one IaC fixture file.
func dirHasFixture(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && isFixtureFile(e.Name()) {
			return true
		}
	}
	return false
}

func indent(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = fmt.Sprintf("    %s", s)
	}
	return out
}
