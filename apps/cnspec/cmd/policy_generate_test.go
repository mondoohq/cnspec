// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.mondoo.com/cnspec/v13/internal/bundle"
	"go.mondoo.com/cnspec/v13/internal/generate"
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
