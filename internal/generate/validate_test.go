// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql/types"
)

// compileValidatorFor builds the real compile validator, skipping the test when
// the provider whose resources it needs is not installed. CI runs the Go tests
// with PROVIDERS_PATH pointed at an empty directory, so a test over the schema
// has to say out loud that it checked nothing rather than pass on an empty run.
func compileValidatorFor(t *testing.T, provider string) Validator {
	t.Helper()
	v, err := NewCompileValidator()
	if err != nil {
		t.Skipf("no compiler available: %v", err)
	}
	if checker, ok := v.(ProviderChecker); ok {
		if err := checker.CheckProvider(provider); err != nil {
			t.Skipf("provider %q not installed, nothing to check: %v", provider, err)
		}
	}
	return v
}

// TestCompileValidatorResolvesProps is the measured blocker: the prompt tells the
// agent to reference props.<name>, and the compile gate was called with a nil
// props handler, so it rejected every query that did what the prompt asked. All
// 49 prop-using checks in content/ fail to compile without a handler and all 49
// compile with one.
func TestCompileValidatorResolvesProps(t *testing.T) {
	v := compileValidatorFor(t, "os")

	props := []Prop{{
		Name: "sshdCiphers",
		// no declared type: the shape every prop in content/ has, so the type
		// has to come from compiling this snippet
		Mql: `return ["aes256-ctr", "aes192-ctr"]`,
	}}
	query := `sshd.config.params["Ciphers"].split(",").containsOnly(props.sshdCiphers)`

	if err := v.Validate(ValidationRequest{MQL: query, Props: props, Provider: "os"}); err != nil {
		t.Fatalf("a correct prop-using query was rejected: %v", err)
	}

	// and without the props it is genuinely unresolvable, which is what made the
	// missing handler look like a bad answer from the agent
	if err := v.Validate(ValidationRequest{MQL: query, Provider: "os"}); err == nil {
		t.Fatal("expected an unknown-property error when the props are withheld")
	}
}

func TestPropTypeResolution(t *testing.T) {
	cases := []struct {
		name     string
		declared string
		want     types.Type
	}{
		{"llx code passes through", string(types.String), types.String},
		{"human alias", "int", types.Int},
		{"human alias, mixed case", "Bool", types.Bool},
		{"slice", "[]string", types.Array(types.String)},
		{"map", "map[string]int", types.Map(types.String, types.Int)},
		{"unknown is not guessed", "SomeResource", types.NoType},
		{"empty is not guessed", "", types.NoType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parsePropType(tc.declared); got != tc.want {
				t.Fatalf("parsePropType(%q) = %q, want %q", tc.declared, got.Label(), tc.want.Label())
			}
		})
	}
}

// TestCompileValidatorRejectsNonVerdicts covers the truncation shape: a query
// that compiles and returns data rather than a pass/fail answer. `aws.s3.buckets`
// was accepted as a scored check body.
func TestCompileValidatorRejectsNonVerdicts(t *testing.T) {
	v := compileValidatorFor(t, "aws")

	accepted := []string{
		"aws.s3.buckets.all(encryption != empty)",
		`sshd.config { params["PermitRootLogin"] == "no" }`,
		"aws.s3.buckets { encryption != empty }",
	}
	rejected := []string{
		"aws.s3.buckets",          // the truncated answer
		"aws.s3.buckets { name }", // a data query, not a check
		"os.hostname",
		"sshd.config",
	}

	for _, q := range accepted {
		if err := v.Validate(ValidationRequest{MQL: q}); err != nil {
			t.Errorf("a real check body was rejected: %q: %v", q, err)
		}
	}
	for _, q := range rejected {
		err := v.Validate(ValidationRequest{MQL: q})
		if err == nil {
			t.Errorf("a query that answers no verdict was accepted: %q", q)
			continue
		}
		if !strings.Contains(err.Error(), "verdict") {
			t.Errorf("the error should say the query answers no verdict, got %q: %v", q, err)
		}
	}
}

// TestCompileValidatorReportsAMissingProvider pins the honest-failure choice: on
// a machine with no providers installed, NewCompileValidator used to succeed and
// then reject every check with "cannot find resource for identifier 'aws'" —
// three times per check, after three agent invocations, phrased as though the
// generated query were at fault.
func TestCompileValidatorReportsAMissingProvider(t *testing.T) {
	v := &compileValidator{}
	v.installedOnce.Do(func() { v.installed = map[string]bool{"os": true} })

	if err := v.CheckProvider("os"); err != nil {
		t.Fatalf("an installed provider must pass: %v", err)
	}
	if err := v.CheckProvider(""); err != nil {
		t.Fatalf("an unresolved provider is not an error: %v", err)
	}

	err := v.CheckProvider("aws")
	if err == nil {
		t.Fatal("expected an error for a provider that is not installed")
	}
	if !errors.Is(err, ErrValidationUnavailable) {
		t.Fatalf("the error must be marked as 'not validated', got: %v", err)
	}
	if !strings.Contains(err.Error(), "cnspec providers install aws") {
		t.Fatalf("the error should tell the user how to fix it, got: %v", err)
	}

	// and the same answer arrives through Validate, so a caller that never asks
	// still cannot mistake it for a compile failure
	if err := v.Validate(ValidationRequest{MQL: "aws.s3.buckets", Provider: "aws"}); !errors.Is(err, ErrValidationUnavailable) {
		t.Fatalf("Validate must report the missing provider, got: %v", err)
	}
}
