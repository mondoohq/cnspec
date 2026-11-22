// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"context"
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
)

// scriptBackend returns canned MQL when the prompt contains a known marker,
// making harness runs deterministic offline. Real evaluation swaps in a real
// AgentBackend.
type scriptBackend struct {
	replies map[string]string // prompt substring -> MQL
}

func (s scriptBackend) Name() string    { return "script" }
func (s scriptBackend) Available() bool { return true }
func (s scriptBackend) Generate(_ context.Context, t GenTask) (GenResult, error) {
	for marker, mql := range s.replies {
		if strings.Contains(t.Prompt, marker) {
			return GenResult{MQL: mql, Raw: mql}, nil
		}
	}
	return GenResult{}, errors.New("no scripted reply for prompt")
}

// goldenReplies maps each case's intent to a canned "correct" MQL answer. Keying
// by the full intent (which the prompt echoes verbatim) makes matches
// unambiguous. This doubles as a check that every case's wantContains substrings
// are self-consistent with a plausible correct query.
var goldenReplies = map[string]string{
	"S3 buckets must be encrypted":                               "aws.s3.buckets.all(encryption != empty)",
	"EBS volumes must be encrypted by default":                   "aws.ec2.ebsEncryptionByDefault.values.all(_ == true)",
	"CloudTrail must be enabled across all regions":              "aws.cloudtrail.trails.any(isMultiRegionTrail == true)",
	"IAM Access Analyzer must be enabled":                        "aws.iam.accessAnalyzer.analyzers.length > 0",
	"Compute disks must be encrypted with customer-managed keys": "gcp.project.computeService.disks.all(diskEncryptionKey != empty)",
	"Cloud SQL instances must require SSL":                       "gcp.project.sqlService.instances.all(settings.ipConfiguration.requireSsl == true)",
	"Load balancer backend services must have logging enabled":   "gcp.project.computeService.backendServices.all(logConfig != empty)",
	"Storage accounts must not allow public blob access":         "azure.subscription.storage.accounts.all(allowBlobPublicAccess == false)",
	"Managed disks must be encrypted":                            "azure.subscription.compute.disks.all(encryptionType != empty)",
	"SQL servers must enforce Azure AD-only authentication":      "azure.subscription.sql.servers.all(azureAdOnlyAuthentication == true)",
	"SSH root login must be disabled":                            `sshd.config.params["PermitRootLogin"] == "no"`,
	"IP forwarding must be disabled":                             `kernel.parameters["net.ipv4.ip_forward"] == 0`,
	"A file integrity tool must be installed":                    `package("aide").installed == true`,
}

func TestHarness_GoldenCases(t *testing.T) {
	cases, err := LoadCases("testdata/harness_cases.yaml")
	if err != nil {
		t.Fatalf("LoadCases: %v", err)
	}
	if len(cases) < 10 {
		t.Fatalf("expected the expanded corpus (>=10 cases), got %d", len(cases))
	}

	// every case must have a canned reply so the offline run is complete
	for _, c := range cases {
		if _, ok := goldenReplies[c.Intent]; !ok {
			t.Fatalf("no golden reply for case %q (intent %q)", c.Name, c.Intent)
		}
	}

	gen := mustGen(t, Config{Backend: scriptBackend{replies: goldenReplies}, Validator: NoopValidator{}})

	results := RunCases(context.Background(), gen, cases)
	rate, passed, total := PassRate(results)
	if passed != total || rate != 1.0 {
		for _, r := range results {
			if !r.Pass {
				t.Errorf("case %q failed: %s (got %q)", r.Case.Name, r.Reason, r.Got)
			}
		}
		t.Fatalf("expected all %d cases to pass, got %d", total, passed)
	}
}

// TestHarness_GoldenRepliesCompile checks the golden replies themselves. The
// golden run above scores them with NoopValidator, so nothing there noticed that
// three of the thirteen did not compile: they named a map as a bool, indexed a
// resource as a dict, and read a field azure does not have. A reply that cannot
// compile is not the "correct answer" the harness measures a model against.
//
// Compilation needs the provider schemas, and CI deliberately runs the Go tests
// with no providers installed, so this reports what it checked and what it
// skipped rather than passing silently on an empty run.
func TestHarness_GoldenRepliesCompile(t *testing.T) {
	validator, err := NewCompileValidator()
	if err != nil {
		t.Skipf("no compiler available (%v); cannot check the golden replies", err)
	}
	checker, _ := validator.(ProviderChecker)

	cases, err := LoadCases("testdata/harness_cases.yaml")
	if err != nil {
		t.Fatalf("LoadCases: %v", err)
	}

	var checked, skipped int
	for _, c := range cases {
		reply, ok := goldenReplies[c.Intent]
		if !ok {
			t.Errorf("no golden reply for case %q", c.Name)
			continue
		}
		provider, _ := ResolveProvider(c.toCheck())
		if checker != nil && checker.CheckProvider(provider) != nil {
			skipped++
			continue
		}
		checked++
		if err := validator.Validate(ValidationRequest{MQL: reply, Provider: provider, Props: c.Props}); err != nil {
			t.Errorf("golden reply for %q does not validate: %v\n  %s", c.Name, err, reply)
		}
	}

	t.Logf("golden replies: %d compiled, %d skipped (provider not installed)", checked, skipped)
	if checked == 0 {
		t.Skip("no target provider installed; no golden reply was actually checked")
	}
}

func TestHarness_DetectsBadOutput(t *testing.T) {
	// a reply that omits an expected substring must be scored as a failure — this
	// is what lets the harness catch a regression.
	c := Case{
		Name:         "s3",
		Intent:       "S3 buckets must be encrypted",
		WantContains: []string{"aws.s3.buckets", "encryption"},
	}
	backend := scriptBackend{replies: map[string]string{
		"S3 buckets must be encrypted": "aws.s3.buckets.length > 0", // missing "encryption"
	}}
	gen := mustGen(t, Config{Backend: backend, Validator: NoopValidator{}})

	results := RunCases(context.Background(), gen, []Case{c})
	if results[0].Pass {
		t.Fatal("expected the case to fail (missing 'encryption')")
	}
	if !strings.Contains(results[0].Reason, "encryption") {
		t.Fatalf("reason should name the missing substring, got %q", results[0].Reason)
	}
}

func TestHarness_ExactMatch(t *testing.T) {
	c := Case{Name: "x", Intent: "trivial", Want: "a == b"}
	backend := scriptBackend{replies: map[string]string{"trivial": "a  ==   b"}} // whitespace differs
	gen := mustGen(t, Config{Backend: backend, Validator: NoopValidator{}})

	results := RunCases(context.Background(), gen, []Case{c})
	if !results[0].Pass {
		t.Fatalf("exact match should normalize whitespace; got %q, reason %q", results[0].Got, results[0].Reason)
	}
}
