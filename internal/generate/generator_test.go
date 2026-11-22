// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"context"
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
)

// spyValidator records what it was asked to judge.
type spyValidator struct {
	requests []ValidationRequest
}

func (v *spyValidator) Validate(req ValidationRequest) error {
	v.requests = append(v.requests, req)
	return nil
}

// unavailableValidator stands in for "this machine cannot compile-check that
// provider".
type unavailableValidator struct {
	checked []string
}

func (v *unavailableValidator) Validate(ValidationRequest) error { return nil }

func (v *unavailableValidator) CheckProvider(provider string) error {
	v.checked = append(v.checked, provider)
	if provider == "aws" {
		return errors.Mark(errors.New("the \"aws\" provider is not installed"), ErrValidationUnavailable)
	}
	return nil
}

// TestGeneratorGivesTheValidatorItsProps is the engine half of the props fix:
// the check's props have to travel with the query, or the compile gate judges a
// prop-using answer without knowing the props exist.
func TestGeneratorGivesTheValidatorItsProps(t *testing.T) {
	backend := &fakeBackend{avail: true, responses: []string{`{"mql":"sshd.config.params[\"Ciphers\"] == props.ciphers"}`}}
	validator := &spyValidator{}
	gen := mustGen(t, Config{Backend: backend, Validator: validator})

	props := []Prop{{Name: "ciphers", Mql: `return ["aes256-ctr"]`}}
	res := gen.GenerateCheck(context.Background(), Check{
		UID:     "sshd-ciphers",
		Title:   "SSH must use strong ciphers",
		Filters: []string{`asset.family.contains("linux")`},
		Props:   props,
	})
	if res.Action != ActionGenerated {
		t.Fatalf("expected generated, got %s: %v", res.Action, res.Err)
	}
	if len(validator.requests) != 1 {
		t.Fatalf("expected one validation, got %d", len(validator.requests))
	}
	req := validator.requests[0]
	if len(req.Props) != 1 || req.Props[0].Name != "ciphers" {
		t.Fatalf("the check's props did not reach the validator: %+v", req.Props)
	}
	if req.Provider != "os" {
		t.Fatalf("the target provider did not reach the validator, got %q", req.Provider)
	}
}

// TestGeneratorRejectsEmptyMQL keeps the contract where the contract is
// documented. The backend happens to reject an empty answer today; the engine
// must not depend on that, because ActionGenerated with no MQL is written into
// the bundle as an empty check body.
func TestGeneratorRejectsEmptyMQL(t *testing.T) {
	backend := &emptyAnswerBackend{}
	gen := mustGen(t, Config{Backend: backend, MaxAttempts: 2})

	res := gen.GenerateCheck(context.Background(), Check{UID: "u", Title: "something"})
	if res.Action == ActionGenerated {
		t.Fatalf("an empty answer was reported as generated (mql %q)", res.MQL)
	}
	if res.MQL != "" {
		t.Fatalf("expected no MQL, got %q", res.MQL)
	}
	if backend.calls != 2 {
		t.Fatalf("an empty answer should be retried like any other bad answer, got %d calls", backend.calls)
	}
}

// emptyAnswerBackend returns whitespace and no error, the case a backend that
// does not check for itself would let through.
type emptyAnswerBackend struct{ calls int }

func (b *emptyAnswerBackend) Name() string    { return "empty" }
func (b *emptyAnswerBackend) Available() bool { return true }
func (b *emptyAnswerBackend) Generate(context.Context, GenTask) (GenResult, error) {
	b.calls++
	return GenResult{MQL: "   \n"}, nil
}

// TestValidateWithoutAValidator: "nothing checked this" must not look like
// "this is valid" to the caller accepting hand-edited MQL.
func TestValidateWithoutAValidator(t *testing.T) {
	gen := mustGen(t, Config{Backend: &fakeBackend{avail: true}})

	err := gen.Validate("aws.s3.buckets.all(encryption != empty)")
	if err == nil {
		t.Fatal("Validate returned nil with no validator configured; the caller cannot tell that nothing checked the query")
	}
	if !errors.Is(err, ErrNoValidator) || !errors.Is(err, ErrValidationUnavailable) {
		t.Fatalf("expected ErrNoValidator (and the unavailable marker), got %v", err)
	}

	// with a validator, it delegates and carries the props
	spy := &spyValidator{}
	gen = mustGen(t, Config{Backend: &fakeBackend{avail: true}, Validator: spy})
	if err := gen.Validate("x != empty", Prop{Name: "p"}); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(spy.requests) != 1 || len(spy.requests[0].Props) != 1 {
		t.Fatalf("props did not reach the validator: %+v", spy.requests)
	}
}

// TestGenerateReportsTheCancelledTail: a caller rendering one row per check gets
// a result for every check it handed over. Dropping the tail leaves those rows
// pending forever, which is inconsistent with the per-check cancel result
// GenerateCheck already returns.
func TestGenerateReportsTheCancelledTail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	backend := &cancellingBackend{cancel: cancel}
	gen := mustGen(t, Config{Backend: backend, MaxAttempts: 1})

	checks := []Check{
		{UID: "first", Title: "one"},
		{UID: "second", Title: "two"},
		{UID: "third", Title: "three"},
	}
	var progressed []string
	results := gen.Generate(ctx, checks, func(r Result) { progressed = append(progressed, r.UID) })

	if len(results) != len(checks) {
		t.Fatalf("expected a result per check, got %d of %d", len(results), len(checks))
	}
	if len(progressed) != len(checks) {
		t.Fatalf("expected progress for every check, got %v", progressed)
	}
	for _, r := range results[1:] {
		if r.Action != ActionFailed || r.Reason != "cancelled" {
			t.Fatalf("check %q after cancellation should report cancelled, got %s/%q", r.UID, r.Action, r.Reason)
		}
	}
}

// cancellingBackend cancels the run while answering the first check.
type cancellingBackend struct {
	cancel func()
	calls  int
}

func (b *cancellingBackend) Name() string    { return "cancelling" }
func (b *cancellingBackend) Available() bool { return true }
func (b *cancellingBackend) Generate(context.Context, GenTask) (GenResult, error) {
	b.calls++
	b.cancel()
	return GenResult{MQL: "some.query != empty"}, nil
}

// TestGeneratorAsksBeforeSpendingAnAgentCall: an environment that cannot judge
// the answer is reported before the agent runs, not discovered afterwards as
// three compile errors about an unknown resource.
func TestGeneratorAsksBeforeSpendingAnAgentCall(t *testing.T) {
	backend := &fakeBackend{avail: true, responses: []string{`{"mql":"aws.s3.buckets.all(encryption != empty)"}`}}
	validator := &unavailableValidator{}
	gen := mustGen(t, Config{Backend: backend, Validator: validator, MaxAttempts: 3})

	res := gen.GenerateCheck(context.Background(), Check{
		UID:     "s3",
		Title:   "S3 buckets must be encrypted",
		Filters: []string{`asset.platform == "aws"`},
	})
	if res.Action != ActionFailed {
		t.Fatalf("expected failure, got %s", res.Action)
	}
	if backend.calls != 0 {
		t.Fatalf("the agent should not be invoked when the answer cannot be judged, got %d calls", backend.calls)
	}
	if !strings.Contains(res.Reason, "not installed") {
		t.Fatalf("the reason should name the missing provider, got %q", res.Reason)
	}

	// a provider it can validate is generated normally
	res = gen.GenerateCheck(context.Background(), Check{
		UID:     "sshd",
		Title:   "SSH root login must be disabled",
		Filters: []string{`asset.family.contains("linux")`},
	})
	if res.Action != ActionGenerated {
		t.Fatalf("expected generated for a checkable provider, got %s: %v", res.Action, res.Err)
	}
}

// TestGeneratorStopsRetryingWhenValidationIsUnavailable: an unavailable
// validator is not a wrong answer, so it must not be reported as "failed after 3
// attempts" with three agent calls spent on it.
func TestGeneratorStopsRetryingWhenValidationIsUnavailable(t *testing.T) {
	backend := &fakeBackend{avail: true, responses: []string{`{"mql":"some.query != empty"}`}}
	gen := mustGen(t, Config{Backend: backend, Validator: brokenValidator{}, MaxAttempts: 3})

	res := gen.GenerateCheck(context.Background(), Check{UID: "u", Title: "t"})
	if res.Action != ActionFailed {
		t.Fatalf("expected failure, got %s", res.Action)
	}
	if backend.calls != 1 {
		t.Fatalf("expected a single attempt, got %d", backend.calls)
	}
	if strings.Contains(res.Reason, "attempts") {
		t.Fatalf("the reason should explain that nothing could judge the query, got %q", res.Reason)
	}
}

type brokenValidator struct{}

func (brokenValidator) Validate(ValidationRequest) error {
	return errors.Mark(errors.New("schema went away"), ErrValidationUnavailable)
}

// rejectingValidator fails every query, as the compile gate does for an answer
// with a typo the agent repeats on every attempt.
type rejectingValidator struct{}

func (rejectingValidator) Validate(ValidationRequest) error {
	return errors.New("cannot find field 'paramz'")
}

// TestGeneratorReturnsTheLastFailedCandidate: when every attempt fails the gate,
// the caller still gets the last query the agent wrote, so the authoring pane
// can offer it for a hand fix instead of another billed run.
func TestGeneratorReturnsTheLastFailedCandidate(t *testing.T) {
	backend := &fakeBackend{avail: true, responses: []string{
		`{"mql":"sshd.config.paramz[\"A\"] == 1"}`,
		`{"mql":"sshd.config.paramz[\"PermitRootLogin\"] == \"no\""}`,
	}}
	gen := mustGen(t, Config{Backend: backend, Validator: rejectingValidator{}, MaxAttempts: 2})

	res := gen.GenerateCheck(context.Background(), Check{UID: "u", Title: "SSH root login disabled"})
	if res.Action != ActionFailed {
		t.Fatalf("expected failed, got %s", res.Action)
	}
	if want := `sshd.config.paramz["PermitRootLogin"] == "no"`; res.MQL != want {
		t.Fatalf("failed result MQL = %q, want the last candidate %q", res.MQL, want)
	}
	if !strings.Contains(res.Reason, "paramz") {
		t.Fatalf("the gate's error should be in the reason, got %q", res.Reason)
	}
}

// TestGeneratorGuessesTheProviderWithoutFilters: a check written as a title and
// description has no filters, and without a provider the up-front install check
// is skipped, so a missing provider costs every attempt.
func TestGeneratorGuessesTheProviderWithoutFilters(t *testing.T) {
	backend := &fakeBackend{avail: true, responses: []string{`{"mql":"sshd.config.params[\"PermitRootLogin\"] == \"no\""}`}}
	validator := &unavailableValidator{}
	gen := mustGen(t, Config{Backend: backend, Validator: validator})

	res := gen.GenerateCheck(context.Background(), Check{UID: "ssh-no-root", Title: "SSH must not permit root login"})
	if res.Provider != "os" {
		t.Fatalf("provider = %q, want os guessed from the title", res.Provider)
	}
	if len(validator.checked) != 1 || validator.checked[0] != "os" {
		t.Fatalf("the guessed provider was not checked before the agent ran: %v", validator.checked)
	}

	// and a missing guessed provider costs no agent call
	backend.calls = 0
	res = gen.GenerateCheck(context.Background(), Check{UID: "s3", Title: "S3 buckets must be encrypted"})
	if res.Action != ActionFailed || backend.calls != 0 {
		t.Fatalf("expected an up-front failure with no agent call, got %s after %d call(s)", res.Action, backend.calls)
	}
}

// TestGeneratorReportsEachAttempt: the front end is told before every agent
// call, with the reason the previous answer was rejected, so a long run shows
// what it is doing.
func TestGeneratorReportsEachAttempt(t *testing.T) {
	backend := &fakeBackend{avail: true, responses: []string{`{"mql":"sshd.config.paramz[\"A\"] == 1"}`}}
	var seen []Attempt
	gen := mustGen(t, Config{Backend: backend, Validator: rejectingValidator{}, MaxAttempts: 3,
		OnAttempt: func(a Attempt) { seen = append(seen, a) }})

	gen.GenerateCheck(context.Background(), Check{UID: "u", Title: "SSH root login disabled"})
	if len(seen) != 3 {
		t.Fatalf("got %d attempt reports, want 3", len(seen))
	}
	if seen[0].N != 1 || seen[0].Max != 3 || seen[0].Retrying != nil {
		t.Errorf("first attempt = %+v, want N=1 Max=3 and no retry reason", seen[0])
	}
	if seen[2].N != 3 || seen[2].Retrying == nil || !strings.Contains(seen[2].Retrying.Error(), "paramz") {
		t.Errorf("third attempt = %+v, want N=3 with the gate's error as the reason", seen[2])
	}
}

// resolvingValidator accepts every query and names its provider.
type resolvingValidator struct{}

func (resolvingValidator) Validate(ValidationRequest) error { return nil }
func (resolvingValidator) ProviderForQuery(string) string   { return "os" }

// TestGeneratorTakesTheProviderFromTheQuery: when neither filters nor wording
// name a target, the validated query does.
func TestGeneratorTakesTheProviderFromTheQuery(t *testing.T) {
	backend := &fakeBackend{avail: true, responses: []string{`{"mql":"file(\"/etc/shadow\").permissions.other_readable == false"}`}}
	gen := mustGen(t, Config{Backend: backend, Validator: resolvingValidator{}})
	res := gen.GenerateCheck(context.Background(), Check{UID: "shadow", Title: "/etc/shadow must not be world readable"})
	if res.Action != ActionGenerated || res.Provider != "os" {
		t.Fatalf("got %s with provider %q, want generated with os from the query", res.Action, res.Provider)
	}
}
