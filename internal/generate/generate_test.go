// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"context"
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
)

// --- fakes ---------------------------------------------------------------

type fakeBackend struct {
	name      string
	avail     bool
	responses []string
	calls     int
	forcedErr error
}

func (f *fakeBackend) Name() string    { return f.name }
func (f *fakeBackend) Available() bool { return f.avail }
func (f *fakeBackend) Generate(_ context.Context, _ GenTask) (GenResult, error) {
	if f.forcedErr != nil {
		return GenResult{}, f.forcedErr
	}
	idx := f.calls
	if idx >= len(f.responses) {
		idx = len(f.responses) - 1
	}
	f.calls++
	resp := f.responses[idx]
	res := parseResponse(resp)
	res.Raw = resp
	if res.MQL == "" {
		return res, errors.New("no mql")
	}
	return res, nil
}

type fakeValidator struct {
	rejectN int
	calls   int
}

func (v *fakeValidator) Validate(string) error {
	v.calls++
	if v.calls <= v.rejectN {
		return errors.New("does not compile")
	}
	return nil
}

// --- tests ---------------------------------------------------------------

func TestResolveProvider(t *testing.T) {
	cases := []struct {
		name     string
		check    Check
		provider string
	}{
		{"platform literal", Check{Filters: []string{`asset.platform == "aws"`}}, "aws"},
		{"eks platform", Check{Filters: []string{`asset.platform == "aws-eks-cluster"`}}, "aws"},
		{"linux family", Check{Filters: []string{`asset.family.contains("ubuntu")`}}, "os"},
		{"windows", Check{Filters: []string{`asset.platform == "windows"`}}, "os"},
		{"resource prefix", Check{Filters: []string{`aws.ec2.instances.any(_.state == "running")`}}, "aws"},
		{"uid fallback", Check{UID: "mondoo-gcp-security-foo"}, "gcp"},
		{"unknown", Check{Filters: []string{`asset.platform == "bogus"`}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := ResolveProvider(tc.check)
			if got != tc.provider {
				t.Fatalf("ResolveProvider = %q, want %q", got, tc.provider)
			}
		})
	}
}

func TestParseResponse(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			"fenced json",
			"Here you go:\n```json\n{\"mql\": \"aws.s3.buckets.all(encryption != empty)\"}\n```\n",
			"aws.s3.buckets.all(encryption != empty)",
		},
		{
			"bare json object",
			"prefix {\"mql\": \"users.all(uid >= 0)\", \"explanation\": \"x\"} suffix",
			"users.all(uid >= 0)",
		},
		{
			"json with braces in string",
			"```json\n{\"mql\": \"a.where(b == \\\"{x}\\\")\"}\n```",
			`a.where(b == "{x}")`,
		},
		{
			"fenced mql fallback",
			"I could not JSON but here:\n```mql\nsshd.config.params[\"X\"] == \"no\"\n```",
			`sshd.config.params["X"] == "no"`,
		},
		{
			"empty",
			"I cannot help with that.",
			"",
		},
		{
			// M3: agent echoes an example envelope first, real answer second — last wins
			"echoed example then answer",
			"Example:\n```json\n{\"mql\": \"users.all(uid >= 0)\"}\n```\nMy answer:\n```json\n{\"mql\": \"aws.s3.buckets.all(encryption != empty)\"}\n```",
			"aws.s3.buckets.all(encryption != empty)",
		},
		{
			// M4: an explanation-only (empty mql) envelope must not shadow the real one
			"empty-mql envelope then answer",
			"```json\n{\"explanation\": \"thinking...\"}\n```\n```json\n{\"mql\": \"sshd.config.params[\\\"X\\\"] == \\\"no\\\"\"}\n```",
			`sshd.config.params["X"] == "no"`,
		},
		{
			// M3: an explanation-only json fence must NOT be returned as raw MQL
			"explanation-only json is not mql",
			"```json\n{\"explanation\": \"I could not produce a query\"}\n```",
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseResponse(tc.raw)
			if got.MQL != tc.want {
				t.Fatalf("parseResponse MQL = %q, want %q", got.MQL, tc.want)
			}
		})
	}
}

func TestCorpusSearch(t *testing.T) {
	corpus := NewCorpus([]Example{
		{UID: "a", Title: "S3 buckets must be encrypted", Desc: "server-side encryption", Mql: "aws.s3.buckets.all(encryption != empty)", Provider: "aws"},
		{UID: "b", Title: "Compute disks must be encrypted", Desc: "disk encryption at rest", Mql: "gcp.compute.disks.all(diskEncryptionKey != empty)", Provider: "gcp"},
		{UID: "c", Title: "SSH root login disabled", Desc: "harden sshd", Mql: "sshd.config.params[\"PermitRootLogin\"] == \"no\"", Provider: "os"},
	})

	got := corpus.Search("bucket encryption enabled", "aws", 2)
	if len(got) == 0 {
		t.Fatal("expected results")
	}
	if got[0].UID != "a" {
		t.Fatalf("expected s3 example first, got %q", got[0].UID)
	}

	// provider bias: gcp query should surface the gcp example first even though
	// "encryption" matches both.
	gotGCP := corpus.Search("disk encryption", "gcp", 1)
	if len(gotGCP) != 1 || gotGCP[0].UID != "b" {
		t.Fatalf("expected gcp example, got %+v", gotGCP)
	}

	// stemming: singular "bucket" query should still match the plural "buckets"
	// example even without a provider hint.
	gotStem := corpus.Search("bucket encrypt", "", 1)
	if len(gotStem) != 1 || gotStem[0].UID != "a" {
		t.Fatalf("expected stemmed match on s3 example, got %+v", gotStem)
	}
}

func TestGeneratorSkip(t *testing.T) {
	gen := mustGen(t, Config{Backend: &fakeBackend{avail: true, responses: []string{`{"mql":"x"}`}}})

	// existing mql, no force -> skipped
	res := gen.GenerateCheck(context.Background(), Check{UID: "u", Title: "t", Mql: "already"})
	if res.Action != ActionSkipped {
		t.Fatalf("expected skipped, got %s", res.Action)
	}
	if res.MQL != "already" {
		t.Fatalf("skip should preserve mql, got %q", res.MQL)
	}

	// no intent -> skipped
	res = gen.GenerateCheck(context.Background(), Check{UID: "u"})
	if res.Action != ActionSkipped {
		t.Fatalf("expected skipped for no intent, got %s", res.Action)
	}

	// variant parent -> skipped even with intent and no mql
	res = gen.GenerateCheck(context.Background(), Check{UID: "u", Title: "parent", HasVariants: true})
	if res.Action != ActionSkipped {
		t.Fatalf("expected variant parent skipped, got %s", res.Action)
	}
}

func TestGeneratorGenerate(t *testing.T) {
	backend := &fakeBackend{avail: true, responses: []string{`{"mql":"aws.s3.buckets.all(encryption != empty)","explanation":"checks encryption"}`}}
	gen := mustGen(t, Config{Backend: backend, Explain: true})

	res := gen.GenerateCheck(context.Background(), Check{
		UID:     "s3-enc",
		Title:   "S3 buckets must be encrypted",
		Filters: []string{`asset.platform == "aws"`},
	})
	if res.Action != ActionGenerated {
		t.Fatalf("expected generated, got %s: %v", res.Action, res.Err)
	}
	if res.Provider != "aws" {
		t.Fatalf("expected provider aws, got %q", res.Provider)
	}
	if !strings.Contains(res.MQL, "encryption") {
		t.Fatalf("unexpected mql: %q", res.MQL)
	}
	if res.Explanation == "" {
		t.Fatal("expected explanation")
	}
}

func TestGeneratorValidationRetry(t *testing.T) {
	// first response invalid per validator, second accepted
	backend := &fakeBackend{avail: true, responses: []string{
		`{"mql":"bad query"}`,
		`{"mql":"good.query != empty"}`,
	}}
	validator := &fakeValidator{rejectN: 1}
	gen := mustGen(t, Config{Backend: backend, Validator: validator, MaxAttempts: 3})

	res := gen.GenerateCheck(context.Background(), Check{UID: "u", Title: "something"})
	if res.Action != ActionGenerated {
		t.Fatalf("expected generated after retry, got %s: %v", res.Action, res.Err)
	}
	if res.MQL != "good.query != empty" {
		t.Fatalf("expected second response, got %q", res.MQL)
	}
	if backend.calls != 2 {
		t.Fatalf("expected 2 backend calls, got %d", backend.calls)
	}
}

func TestGeneratorValidationExhausted(t *testing.T) {
	backend := &fakeBackend{avail: true, responses: []string{`{"mql":"always bad"}`}}
	validator := &fakeValidator{rejectN: 100}
	gen := mustGen(t, Config{Backend: backend, Validator: validator, MaxAttempts: 2})

	res := gen.GenerateCheck(context.Background(), Check{UID: "u", Title: "t"})
	if res.Action != ActionFailed {
		t.Fatalf("expected failed, got %s", res.Action)
	}
	if backend.calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", backend.calls)
	}
}

type fakeRunner struct {
	value    bool
	resolved bool
	err      error
}

func (r fakeRunner) Run(string) (bool, bool, error) { return r.value, r.resolved, r.err }

func TestExecuteValidator(t *testing.T) {
	cases := []struct {
		name    string
		runner  fakeRunner
		wantErr bool
	}{
		{"resolves true", fakeRunner{value: true, resolved: true}, false},
		{"resolves false (still valid)", fakeRunner{value: false, resolved: true}, false},
		{"null result rejected", fakeRunner{resolved: false}, true},
		{"exec error rejected", fakeRunner{err: errors.New("boom")}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := NewExecuteValidator(tc.runner, NoopValidator{})
			if err != nil {
				t.Fatalf("NewExecuteValidator: %v", err)
			}
			err = v.Validate("some.query != empty")
			if tc.wantErr != (err != nil) {
				t.Fatalf("Validate err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestExecuteValidatorCompilesFirst(t *testing.T) {
	// a compile failure must short-circuit before the runner is consulted
	rejectCompile := failValidator{}
	ran := &countingRunner{}
	v, err := NewExecuteValidator(ran, rejectCompile)
	if err != nil {
		t.Fatalf("NewExecuteValidator: %v", err)
	}
	if err := v.Validate("x"); err == nil {
		t.Fatal("expected compile error")
	}
	if ran.calls != 0 {
		t.Fatalf("runner should not be called when compile fails, got %d calls", ran.calls)
	}
}

type failValidator struct{}

func (failValidator) Validate(string) error { return errors.New("does not compile") }

type countingRunner struct{ calls int }

func (r *countingRunner) Run(string) (bool, bool, error) {
	r.calls++
	return true, true, nil
}

func TestBackendSelection(t *testing.T) {
	if _, err := Backend("does-not-exist"); err == nil {
		t.Fatal("expected error for unknown agent")
	}
	names := BackendNames()
	if len(names) != 4 || names[0] != "claude" {
		t.Fatalf("unexpected backend names: %v", names)
	}
}

func mustGen(t *testing.T, cfg Config) *Generator {
	t.Helper()
	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return g
}
