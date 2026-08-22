// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package ocsf

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratedTypesAreCurrent re-runs the generator and compares its output with
// what is committed. It fails when gen.yaml or a schema changed without
// `go generate ./cli/reporter/ocsf/...` being run, which would otherwise only
// surface as types that quietly disagree with the schema they claim to follow.
func TestGeneratedTypesAreCurrent(t *testing.T) {
	dir := t.TempDir()

	cmd := exec.Command("go", "run", "./internal/gen", "-o", dir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "generator failed: %s", out)

	for _, name := range []string{"types.gen.go", "enums.gen.go"} {
		want, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		got, err := os.ReadFile(name)
		require.NoError(t, err)
		assert.Equal(t, string(want), string(got),
			"%s is out of date, run: go generate ./cli/reporter/ocsf/...", name)
	}
}
