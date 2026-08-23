// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/internal/bundle"
	"go.mondoo.com/cnspec/internal/generate"
)

// stubAgent writes an executable shell script that emits a fixed JSON envelope,
// standing in for a coding-agent CLI so the command can be exercised offline.
func stubAgent(t *testing.T, dir, mql string) string {
	t.Helper()
	path := filepath.Join(dir, "stub-agent.sh")
	script := "#!/usr/bin/env bash\ncat <<'EOF'\n```json\n{\"mql\": \"" + mql + "\"}\n```\nEOF\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	return path
}

const genCmdFixture = `queries:
  - uid: empty-check
    title: Empty check that needs mql
    filters: asset.platform == "aws"
    docs:
      desc: All buckets must be encrypted.
  - uid: has-check
    title: Already implemented
    mql: |
      existing.value == 1
  - uid: variant-parent
    title: Parent intent for variants
    docs:
      desc: The intent lives here, not on the leaves.
    variants:
      - uid: variant-parent-aws
  - uid: variant-parent-aws
    filters: asset.platform == "aws"
`

func TestPolicyGenerate_WriteBack(t *testing.T) {
	dir := t.TempDir()
	agent := stubAgent(t, dir, "generated.value == true")
	t.Setenv("CNSPEC_AGENT_CLAUDE_BIN", agent)

	file := filepath.Join(dir, "policy.mql.yaml")
	if err := os.WriteFile(file, []byte(genCmdFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	backend, err := generate.Backend("claude")
	if err != nil {
		t.Fatalf("backend: %v", err)
	}
	gen, err := generate.New(generate.Config{
		Backend:   backend,
		Validator: generate.NoopValidator{}, // no providers needed offline
	})
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}

	generated, failed, err := generateForFile(context.Background(), gen, file, false, false, true, "", false)
	if err != nil {
		t.Fatalf("generateForFile: %v", err)
	}
	if failed != 0 {
		t.Fatalf("expected 0 failed, got %d", failed)
	}
	// empty-check and variant-parent-aws (leaf) should generate; parent and
	// has-check should not.
	if generated != 2 {
		t.Fatalf("expected 2 generated, got %d", generated)
	}

	out, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	b, err := bundle.ParseYaml(out)
	if err != nil {
		t.Fatalf("parse back: %v", err)
	}
	byUID := map[string]*bundle.Mquery{}
	for _, q := range bundle.AllQueries(b) {
		byUID[q.Uid] = q
	}

	if got := byUID["empty-check"].Mql; got != "generated.value == true" {
		t.Errorf("empty-check mql = %q, want generated", got)
	}
	if got := byUID["variant-parent-aws"].Mql; got != "generated.value == true" {
		t.Errorf("variant leaf mql = %q, want generated (inherited intent)", got)
	}
	if got := strings.TrimSpace(byUID["has-check"].Mql); got != "existing.value == 1" {
		t.Errorf("has-check mql changed to %q, want untouched", got)
	}
	if got := strings.TrimSpace(byUID["variant-parent"].Mql); got != "" {
		t.Errorf("variant parent got mql %q, want empty", got)
	}
}

func TestPolicyGenerate_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	agent := stubAgent(t, dir, "fresh.value == true")
	t.Setenv("CNSPEC_AGENT_CLAUDE_BIN", agent)

	file := filepath.Join(dir, "policy.mql.yaml")
	src := "queries:\n  - uid: c1\n    title: A check\n    filters: asset.platform == \"aws\"\n    mql: old.value == 1\n"
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	backend, _ := generate.Backend("claude")
	gen, _ := generate.New(generate.Config{Backend: backend, Validator: generate.NoopValidator{}, Force: true})

	generated, _, err := generateForFile(context.Background(), gen, file, true, false, true, "", false)
	if err != nil {
		t.Fatalf("generateForFile: %v", err)
	}
	if generated != 1 {
		t.Fatalf("expected 1 generated with --force, got %d", generated)
	}
	out, _ := os.ReadFile(file)
	if !strings.Contains(string(out), "fresh.value == true") {
		t.Fatalf("force did not overwrite; file:\n%s", out)
	}
}

// TestPolicyGenerate_StdoutPassThrough pins that stdout mode always emits the
// bundle. `cnspec policy generate in.yaml > out.yaml` with nothing to generate
// used to print nothing at all, leaving a 0-byte out.yaml and exit 0 — so a
// scripted `... && mv out.yaml in.yaml` destroyed the bundle.
func TestPolicyGenerate_StdoutPassThrough(t *testing.T) {
	dir := t.TempDir()
	agent := stubAgent(t, dir, "unused.value == true")
	t.Setenv("CNSPEC_AGENT_CLAUDE_BIN", agent)

	file := filepath.Join(dir, "policy.mql.yaml")
	// every check already has mql, so there is nothing to generate
	src := "queries:\n  - uid: c1\n    title: A check\n    filters: asset.platform == \"aws\"\n    mql: old.value == 1\n"
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	backend, _ := generate.Backend("claude")
	gen, _ := generate.New(generate.Config{Backend: backend, Validator: generate.NoopValidator{}})

	stdout := captureStdout(t, func() {
		generated, failed, err := generateForFile(context.Background(), gen, file, false, false, false, "", false)
		if err != nil {
			t.Fatalf("generateForFile: %v", err)
		}
		if generated != 0 || failed != 0 {
			t.Fatalf("expected nothing to generate, got %d generated / %d failed", generated, failed)
		}
	})

	if stdout == "" {
		t.Fatal("stdout mode emitted nothing; a redirected run would truncate the caller's file")
	}
	if stdout != src {
		t.Errorf("stdout is not the input bundle:\ngot:\n%s\nwant:\n%s", stdout, src)
	}
}

// captureStdout swaps os.Stdout for a pipe while fn runs. fmt.Print resolves
// os.Stdout at call time, so this catches what the command writes.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var sb strings.Builder
		io.Copy(&sb, r) //nolint:errcheck // the pipe read ends when the writer closes
		done <- sb.String()
	}()

	fn()

	os.Stdout = orig
	w.Close()
	out := <-done
	r.Close()
	return out
}

// TestPolicyGenerate_InteractiveRejectsInPlace pins that a flag the guided flow
// cannot honour is refused rather than silently ignored. It is reachable without
// typing -i: a bare tty run turns the wizard on by itself.
func TestPolicyGenerate_InteractiveRejectsInPlace(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().AddFlagSet(policyGenerateCmd.Flags())
	if err := cmd.Flags().Set("interactive", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("in-place", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		policyGenerateCmd.Flags().Set("interactive", "false") //nolint:errcheck // restoring shared flag state
		policyGenerateCmd.Flags().Set("in-place", "false")    //nolint:errcheck // restoring shared flag state
	})

	err := runPolicyGenerate(cmd, nil)
	if err == nil {
		t.Fatal("--in-place with --interactive was accepted; want an error")
	}
	if !strings.Contains(err.Error(), "--in-place") {
		t.Errorf("error should name the flag: %v", err)
	}
}

// TestBundlePropsCarriesDefiningQuery pins the wiring that makes prop-typing
// work at all. A prop resolves `props.<name>` for the compiler only if it has a
// type, and content props almost never declare `type:` — 80 prop-using queries
// in content/ and the common shape is a bare `mql: '12'`. The type therefore has
// to come from compiling the prop's own defining query, so dropping Mql here
// leaves every prop typed `any` and every comparison against one uncompilable.
func TestBundlePropsCarriesDefiningQuery(t *testing.T) {
	q := &bundle.Mquery{
		Uid: "check-with-props",
		Props: []*bundle.Property{
			{Uid: "minLength", Mql: "12", Title: "Minimum password length"},
			{Uid: "typed", Mql: "\"x\"", Type: "string", Desc: "declared type wins"},
		},
	}

	props := bundleProps(q)
	if len(props) != 2 {
		t.Fatalf("got %d props, want 2", len(props))
	}
	if props[0].Name != "minLength" || props[0].Mql != "12" {
		t.Errorf("prop without a declared type lost its defining query: %+v", props[0])
	}
	// the title is the only human summary an authored prop usually has
	if props[0].Desc != "Minimum password length" {
		t.Errorf("desc fallback to title broke: %q", props[0].Desc)
	}
	if props[1].Type != "string" || props[1].Mql != "\"x\"" {
		t.Errorf("declared type and defining query must both survive: %+v", props[1])
	}
}

// TestPropUsingContentChecksValidate walks every prop-using check in content/
// and compiles it through the same path `cnspec policy generate` uses to gate a
// generated query. It is the end-to-end proof for prop typing: the props a
// check declares must reach the compiler carrying enough type information to
// resolve `props.<name>`.
//
// Whether a check can say anything is decided by asking the provider registry
// what is installed, never by inspecting the compile error. Both failures look
// alike from the outside — a missing provider and an untyped prop each end in
// "does not compile" — so a test that inferred the environment from the error
// would quietly reclassify a real regression as "provider not installed" and
// skip. It did exactly that when first written.
//
// Checks whose provider is installed are held strictly: they must compile. The
// rest are unverifiable here and are counted, so a run that proved nothing
// cannot be mistaken for a run that passed.
func TestPropUsingContentChecksValidate(t *testing.T) {
	v, err := generate.NewCompileValidator()
	if err != nil {
		t.Skipf("no validator available: %v", err)
	}
	checker, ok := v.(generate.ProviderChecker)
	if !ok {
		t.Skip("validator cannot report which providers are installed")
	}

	files, _ := filepath.Glob(filepath.Join("../../../content", "*.mql.yaml"))
	if len(files) == 0 {
		t.Skip("content/ not present")
	}

	var found, verified, unverifiable, failed int
	var failures []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		b, err := bundle.ParseYaml(data)
		if err != nil {
			continue
		}
		for _, q := range b.Queries {
			if q == nil || len(q.Props) == 0 || strings.TrimSpace(q.Mql) == "" {
				continue
			}
			if !strings.Contains(q.Mql, "props.") {
				continue
			}
			found++

			props := bundleProps(q)
			provider, _ := generate.ResolveProvider(generate.Check{
				UID:     q.Uid,
				Filters: bundle.QueryFilterStrings(q),
			})
			// an unresolved provider is not a safe "check it anyway": these are
			// the os/network checks, whose resources are just as absent on a bare
			// box as a named provider's would be.
			if provider == "" || checker.CheckProvider(provider) != nil {
				unverifiable++
				continue
			}

			verified++
			if err := v.Validate(generate.ValidationRequest{MQL: q.Mql, Props: props}); err != nil {
				failed++
				if len(failures) < 5 {
					failures = append(failures, q.Uid+": "+err.Error())
				}
			}
		}
	}

	t.Logf("prop-using checks: %d found, %d verified against installed providers, %d unverifiable",
		found, verified, unverifiable)
	if found == 0 {
		t.Fatal("found 0 prop-using checks — the corpus walk found nothing, so this test asserts nothing")
	}
	if failed > 0 {
		t.Errorf("%d of %d verifiable prop-using checks failed to compile (first %d shown):\n%s",
			failed, verified, len(failures), strings.Join(failures, "\n"))
	}
	if verified == 0 {
		t.Skipf("no provider for any of the %d prop-using checks is installed; prop typing was not exercised", found)
	}
}
