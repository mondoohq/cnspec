// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mondoo.com/cnspec/internal/bundle"
	"go.mondoo.com/cnspec/internal/generate"
)

// cappedBuffer is a concurrency-safe writer that stops growing past a limit. The
// cap matters: a wizard that mishandles EOF spins, and an unbounded buffer turns
// a failing test into an out-of-memory one (the real bug produced 73 MB of
// stderr in six seconds).
type cappedBuffer struct {
	mu  sync.Mutex
	b   strings.Builder
	max int
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.b.Len() < c.max {
		c.b.Write(p)
	}
	return len(p), nil
}

func (c *cappedBuffer) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.b.String()
}

// driveWizard runs the wizard against scripted stdin and fails the test if it
// does not return: every prompt has to end the session at EOF instead of
// answering itself with its default.
func driveWizard(t *testing.T, gen *generate.Generator, opts wizardOpts, script string) (string, error) {
	t.Helper()
	out := &cappedBuffer{max: 1 << 20}
	opts.In = strings.NewReader(script)
	opts.Out = out

	done := make(chan error, 1)
	go func() { done <- runGenerateWizard(context.Background(), gen, opts) }()

	select {
	case err := <-done:
		return out.String(), err
	case <-time.After(10 * time.Second):
		t.Fatalf("wizard did not return within 10s; output so far:\n%s", out.String())
		return "", nil
	}
}

// scriptedAgent writes a stub agent CLI that records every invocation in a
// counter file and prints body. Counting invocations is the point: two of the
// bugs here (EOF picking [r]etry, backing out of [e]dit) show up as extra calls
// to a billed backend, not as a wrong file.
func scriptedAgent(t *testing.T, body string) (bin, counter string) {
	t.Helper()
	dir := t.TempDir()
	bin = filepath.Join(dir, "agent.sh")
	counter = filepath.Join(dir, "calls")
	script := "#!/usr/bin/env bash\necho call >> " + counter + "\ncat <<'AGENTEOF'\n" + body + "\nAGENTEOF\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub agent: %v", err)
	}
	return bin, counter
}

func agentCalls(t *testing.T, counter string) int {
	t.Helper()
	data, err := os.ReadFile(counter)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatalf("read counter: %v", err)
	}
	return len(strings.Fields(string(data)))
}

func testGenerator(t *testing.T, bin string) *generate.Generator {
	t.Helper()
	t.Setenv("CNSPEC_AGENT_CLAUDE_BIN", bin)
	backend, err := generate.Backend("claude")
	if err != nil {
		t.Fatalf("backend: %v", err)
	}
	gen, err := generate.New(generate.Config{
		Backend:     backend,
		Validator:   generate.NoopValidator{}, // no providers needed offline
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	return gen
}

const okAgentReply = "```json\n{\"mql\": \"aws.s3.buckets.all(encryption != empty)\"}\n```"

// --- EOF handling ------------------------------------------------------------

// TestWizardEOFAtRequiredPromptEnds pins the runaway loop: promptRequired used to
// re-print its prompt forever once stdin was at EOF.
func TestWizardEOFAtRequiredPromptEnds(t *testing.T) {
	bin, counter := scriptedAgent(t, okAgentReply)
	gen := testGenerator(t, bin)
	file := filepath.Join(t.TempDir(), "policy.mql.yaml")

	// answers the bundle-file prompt, then stdin ends at the required title
	out, err := driveWizard(t, gen, wizardOpts{File: file}, file+"\n")
	if err != nil {
		t.Fatalf("wizard returned %v, want a clean exit at EOF", err)
	}
	if n := strings.Count(out, "(required)"); n > 1 {
		t.Errorf("re-prompted %d times at EOF; want at most one", n)
	}
	if calls := agentCalls(t, counter); calls != 0 {
		t.Errorf("agent called %d times; EOF before any check must call it none", calls)
	}
}

// TestWizardEOFDoesNotAcceptGeneratedMQL pins the guardrail from
// docs/policy-generation-ux.md: generated MQL is never applied silently. At EOF
// the review prompt used to return its first option — [a]ccept.
func TestWizardEOFDoesNotAcceptGeneratedMQL(t *testing.T) {
	bin, _ := scriptedAgent(t, okAgentReply)
	gen := testGenerator(t, bin)
	file := filepath.Join(t.TempDir(), "policy.mql.yaml")

	// title, desc, provider, filter — then stdin ends at the review prompt
	script := "S3 buckets must be encrypted\n\n\n\n"
	out, err := driveWizard(t, gen, wizardOpts{File: file, FileFromFlag: true}, script)
	if err != nil {
		t.Fatalf("wizard returned %v, want a clean exit at EOF", err)
	}
	if _, statErr := os.Stat(file); statErr == nil {
		data, _ := os.ReadFile(file)
		t.Fatalf("EOF at the review prompt wrote unreviewed MQL to %s:\n%s", file, data)
	}
	if !strings.Contains(out, "added 0 check(s)") {
		t.Errorf("expected a zero-check summary, got:\n%s", out)
	}
}

// TestWizardEOFOnFailedGenerationDoesNotRetry pins the unattended-loop bug: when
// generation failed, the choice prompt returned [r]etry at EOF and the wizard
// re-invoked the agent as fast as it could return (270 invocations in 5s).
func TestWizardEOFOnFailedGenerationDoesNotRetry(t *testing.T) {
	bin, counter := scriptedAgent(t, "this is not a json envelope")
	gen := testGenerator(t, bin)
	file := filepath.Join(t.TempDir(), "policy.mql.yaml")

	script := "S3 buckets must be encrypted\n\n\n\n"
	if _, err := driveWizard(t, gen, wizardOpts{File: file, FileFromFlag: true}, script); err != nil {
		t.Fatalf("wizard returned %v, want a clean exit at EOF", err)
	}
	if calls := agentCalls(t, counter); calls != 1 {
		t.Errorf("agent invoked %d times after a failed generation; EOF must not retry", calls)
	}
}

// --- review loop -------------------------------------------------------------

// TestWizardEditBackoutKeepsCandidate pins that backing out of [e]dit returns to
// the query under review instead of re-entering generation, which cost a second
// billed invocation and swapped the query the user was looking at.
func TestWizardEditBackoutKeepsCandidate(t *testing.T) {
	t.Setenv("EDITOR", "") // force the inline editor, not a real one
	bin, counter := scriptedAgent(t, okAgentReply)
	gen := testGenerator(t, bin)
	file := filepath.Join(t.TempDir(), "policy.mql.yaml")

	// title, desc, provider, filter, [e]dit, empty edit (backs out), [s]kip, no
	script := "S3 buckets must be encrypted\n\n\n\ne\n\ns\nn\n"
	out, err := driveWizard(t, gen, wizardOpts{File: file, FileFromFlag: true}, script)
	if err != nil {
		t.Fatalf("wizard: %v", err)
	}
	if calls := agentCalls(t, counter); calls != 1 {
		t.Errorf("agent invoked %d times; backing out of an edit must not regenerate", calls)
	}
	if n := strings.Count(out, "Generated MQL:"); n != 2 {
		t.Errorf("candidate shown %d times, want 2 (before the edit and after backing out):\n%s", n, out)
	}
}

// TestWizardRendersModelOutputWithoutControlCharacters pins that what the
// reviewer sees is what gets written: an ANSI erase sequence in a generated
// query can repaint the review line, hiding the rest of the query behind
// innocuous-looking text, and FormatBundle strips those on write — so the screen
// showed something the file did not contain.
func TestWizardRendersModelOutputWithoutControlCharacters(t *testing.T) {
	bin, _ := scriptedAgent(t, "```json\n{\"mql\": \"aws.s3.buckets.all(x)\\u001b[2K\\u001b[1G BACKDOOR\"}\n```")
	gen := testGenerator(t, bin)
	file := filepath.Join(t.TempDir(), "policy.mql.yaml")

	// title, desc, provider, filter, [a]ccept, uid, no
	script := "S3 buckets must be encrypted\n\n\n\na\ns3-check\nn\n"
	out, err := driveWizard(t, gen, wizardOpts{File: file, FileFromFlag: true}, script)
	if err != nil {
		t.Fatalf("wizard: %v", err)
	}
	if strings.ContainsRune(out, 0x1b) {
		t.Errorf("review screen rendered a raw escape sequence:\n%q", out)
	}
	if !strings.Contains(out, "BACKDOOR") {
		t.Errorf("the hidden part of the query was not shown to the reviewer:\n%s", out)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	b, err := bundle.ParseYaml(data)
	if err != nil {
		t.Fatalf("parse back: %v", err)
	}
	written := writtenCheck(t, b, "s3-check").Mql
	if !strings.Contains(out, written) {
		t.Errorf("what was written is not what was reviewed:\nwritten: %q\nshown:\n%s", written, out)
	}
}

// --- what the wizard writes --------------------------------------------------

// TestWizardWritesScannableBundle pins that an accepted check lands in a policy
// group. A check that lives only in the top-level `queries:` block lints with a
// query-unassigned warning and then fails `cnspec scan --policy-bundle` with
// "a policy or framework mrn is required" — while the wizard tells the user to
// lint it and commit it.
func TestWizardWritesScannableBundle(t *testing.T) {
	bin, _ := scriptedAgent(t, okAgentReply)
	gen := testGenerator(t, bin)
	file := filepath.Join(t.TempDir(), "aws-s3.mql.yaml")

	script := "S3 buckets must be encrypted\nAll buckets need SSE\n\n\na\ns3-encrypted\nn\n"
	if _, err := driveWizard(t, gen, wizardOpts{File: file, FileFromFlag: true}, script); err != nil {
		t.Fatalf("wizard: %v", err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	b, err := bundle.ParseYaml(data)
	if err != nil {
		t.Fatalf("parse back: %v\n%s", err, data)
	}
	if len(b.Policies) != 1 {
		t.Fatalf("expected exactly one policy, got %d:\n%s", len(b.Policies), data)
	}
	p := b.Policies[0]
	if p.Uid != "aws-s3" || p.Name == "" || p.Version == "" {
		t.Errorf("policy needs a uid, name and version to lint: uid=%q name=%q version=%q", p.Uid, p.Name, p.Version)
	}
	if len(p.Groups) != 1 || len(p.Groups[0].Checks) != 1 {
		t.Fatalf("expected one group with one check, got %+v", p.Groups)
	}
	if got := p.Groups[0].Checks[0].Uid; got != "s3-encrypted" {
		t.Errorf("check uid = %q", got)
	}
	if len(b.Queries) != 0 {
		t.Errorf("check was left dangling in the top-level queries block: %d entries", len(b.Queries))
	}
	q := writtenCheck(t, b, "s3-encrypted")
	if q.Filters == nil {
		t.Error("check lost its asset filter")
	}
	if q.Docs == nil || q.Docs.Desc != "All buckets need SSE" {
		t.Error("check lost its description")
	}
}

// TestWizardJoinsExistingPolicy pins that a check added to a bundle that already
// has a policy joins it, rather than dangling next to it in `queries:`.
func TestWizardJoinsExistingPolicy(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "existing.mql.yaml")
	existing := `policies:
  - uid: existing-policy
    name: Existing policy
    version: 1.0.0
    groups:
      - title: Kubernetes API Server
        filters: |
          asset.family.contains('linux')
          processes.where( executable == /kube-apiserver/ ).list != []
        checks:
          - uid: existing-check
            title: An existing check
            mql: existing.value == 1
`
	if err := os.WriteFile(file, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := bundle.AppendCheck(file, "new-check", "A new check", "", `asset.platform == "aws"`, "aws.s3.buckets.length > 0"); err != nil {
		t.Fatalf("appendCheck: %v", err)
	}

	data, _ := os.ReadFile(file)
	b, err := bundle.ParseYaml(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(b.Policies) != 1 {
		t.Fatalf("expected the check to join the existing policy, got %d policies:\n%s", len(b.Policies), data)
	}
	if len(b.Queries) != 0 {
		t.Errorf("check dangled in the top-level queries block:\n%s", data)
	}
	p := b.Policies[0]
	if len(p.Groups) != 2 {
		t.Fatalf("expected a group of its own, got %d groups:\n%s", len(p.Groups), data)
	}
	// the new check must not inherit the existing group's filter — that group is
	// scoped to hosts running kube-apiserver, which would make an aws check dead
	newGroup := p.Groups[1]
	if newGroup.Filters != nil && len(newGroup.Filters.Items) > 0 {
		t.Errorf("the generated group carries a filter that would gate the check: %+v", newGroup.Filters)
	}
	if len(newGroup.Checks) != 1 || newGroup.Checks[0].Uid != "new-check" {
		t.Errorf("unexpected group contents: %+v", newGroup.Checks)
	}
}

// TestAppendCheck covers the field-level round trip: a scalar filter, a
// description, and a second check landing beside the first.
func TestAppendCheck(t *testing.T) {
	file := filepath.Join(t.TempDir(), "policy.mql.yaml")

	if err := bundle.AppendCheck(file, "c1", "First check", "does a thing", `asset.platform == "aws"`, "aws.s3.buckets.length > 0"); err != nil {
		t.Fatalf("appendCheck 1: %v", err)
	}
	if err := bundle.AppendCheck(file, "c2", "Second check", "", "", "users.all(uid >= 0)"); err != nil {
		t.Fatalf("appendCheck 2: %v", err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	// scalar filter must round-trip (not the map form)
	if !strings.Contains(string(data), `filters: asset.platform == "aws"`) {
		t.Errorf("filter not scalar-serialized:\n%s", data)
	}

	b, err := bundle.ParseYaml(data)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	c1, c2 := writtenCheck(t, b, "c1"), writtenCheck(t, b, "c2")
	if c1.Mql != "aws.s3.buckets.length > 0" {
		t.Errorf("c1 mql = %q", c1.Mql)
	}
	if c1.Docs == nil || c1.Docs.Desc != "does a thing" {
		t.Error("c1 desc missing")
	}
	if c2.Filters != nil {
		t.Error("c2 should have no filter")
	}
}

// TestAppendCheckRejectsDuplicateUID pins that a colliding uid is refused rather
// than appended: two queries with the same uid in one file is a lint error
// (query-uid-unique) the user only discovers later.
func TestAppendCheckRejectsDuplicateUID(t *testing.T) {
	file := filepath.Join(t.TempDir(), "policy.mql.yaml")
	if err := bundle.AppendCheck(file, "c1", "First check", "does a thing", `asset.platform == "aws"`, "aws.s3.buckets.length > 0"); err != nil {
		t.Fatalf("appendCheck 1: %v", err)
	}
	err := bundle.AppendCheck(file, "c1", "Second check", "", "", "users.all(uid >= 0)")
	if err == nil {
		t.Fatal("appending a duplicate uid succeeded; want an error")
	}
	if !strings.Contains(err.Error(), "c1") {
		t.Errorf("error should name the uid: %v", err)
	}

	data, _ := os.ReadFile(file)
	b, _ := bundle.ParseYaml(data)
	if n := len(bundle.QueryUIDs(b)); n != 1 {
		t.Errorf("bundle has %d uids, want 1 (the duplicate must not be written)", n)
	}
}

// TestWizardRePromptsOnUIDCollision pins the interactive half of the same bug:
// the Check UID prompt re-asks instead of writing a bundle that fails lint.
func TestWizardRePromptsOnUIDCollision(t *testing.T) {
	bin, _ := scriptedAgent(t, okAgentReply)
	gen := testGenerator(t, bin)
	file := filepath.Join(t.TempDir(), "policy.mql.yaml")
	if err := bundle.AppendCheck(file, "taken", "Existing", "", `asset.platform == "aws"`, "existing.value == 1"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// title, desc, provider, filter, [a]ccept, colliding uid, free uid, no
	script := "S3 buckets must be encrypted\n\n\n\na\ntaken\nfree-uid\nn\n"
	out, err := driveWizard(t, gen, wizardOpts{File: file, FileFromFlag: true}, script)
	if err != nil {
		t.Fatalf("wizard: %v", err)
	}
	if !strings.Contains(out, "already used") {
		t.Errorf("collision was not reported to the user:\n%s", out)
	}

	data, _ := os.ReadFile(file)
	b, _ := bundle.ParseYaml(data)
	uids := bundle.QueryUIDs(b)
	if !uids["taken"] || !uids["free-uid"] || len(uids) != 2 {
		t.Errorf("unexpected uids %v in:\n%s", uids, data)
	}
}

// TestWizardDryRunWritesNothing pins --dry-run, which the wizard ignored: help
// says "Preview what would be generated without writing anything", and the flag
// is reachable without -i because a bare tty run turns the wizard on itself.
func TestWizardDryRunWritesNothing(t *testing.T) {
	bin, _ := scriptedAgent(t, okAgentReply)
	gen := testGenerator(t, bin)
	file := filepath.Join(t.TempDir(), "policy.mql.yaml")

	script := "S3 buckets must be encrypted\n\n\n\na\ns3-check\nn\n"
	out, err := driveWizard(t, gen, wizardOpts{File: file, FileFromFlag: true, DryRun: true}, script)
	if err != nil {
		t.Fatalf("wizard: %v", err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		data, _ := os.ReadFile(file)
		t.Fatalf("--dry-run wrote %s:\n%s", file, data)
	}
	if !strings.Contains(out, "dry run") || !strings.Contains(out, "s3-check") {
		t.Errorf("dry run did not preview the check:\n%s", out)
	}
}

// TestWizardOutputFlagIsHonoured pins that -o names the bundle instead of being
// ignored while the wizard prompts for a file of its own.
func TestWizardOutputFlagIsHonoured(t *testing.T) {
	bin, _ := scriptedAgent(t, okAgentReply)
	gen := testGenerator(t, bin)
	file := filepath.Join(t.TempDir(), "chosen-by-flag.mql.yaml")

	// no answer for a bundle-file prompt is scripted: there must not be one
	script := "S3 buckets must be encrypted\n\n\n\na\ns3-check\nn\n"
	if _, err := driveWizard(t, gen, wizardOpts{File: file, FileFromFlag: true}, script); err != nil {
		t.Fatalf("wizard: %v", err)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("--output file not written: %v", err)
	}
}

func TestNextFreeUID(t *testing.T) {
	taken := map[string]bool{"a": true, "a-2": true}
	if got := bundle.NextFreeUID("a", taken); got != "a-3" {
		t.Errorf("nextFreeUID = %q, want a-3", got)
	}
	if got := bundle.NextFreeUID("b", taken); got != "b" {
		t.Errorf("nextFreeUID = %q, want b", got)
	}
}

func TestPolicyUIDForFile(t *testing.T) {
	cases := map[string]string{
		"aws-s3.mql.yaml":            "aws-s3",
		"/tmp/policy.mql.yaml":       "policy",
		"x.yaml":                     "generated-policy", // too short to lint
		"My Bundle.mql.yaml":         "my-bundle",
		"/tmp/dir/checks.mql.yml":    "checks",
		"/tmp/dir/checks.mql.foobar": "checks-mql-foobar",
	}
	for in, want := range cases {
		if got := bundle.PolicyUIDForFile(in); got != want {
			t.Errorf("bundle.PolicyUIDForFile(%q) = %q, want %q", in, got, want)
		}
	}
}

func writtenCheck(t *testing.T, b *bundle.Bundle, uid string) *bundle.Mquery {
	t.Helper()
	for _, q := range bundle.AllQueries(b) {
		if q != nil && q.Uid == uid {
			return q
		}
	}
	t.Fatalf("check %q not found in bundle", uid)
	return nil
}
