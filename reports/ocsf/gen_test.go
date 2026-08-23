// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package ocsf

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratedTypesAreCurrent re-runs the generator and compares its output with
// what is committed. It fails when gen.yaml or a schema changed without
// `go generate ./reports/ocsf/...` being run, which would otherwise only
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
			"%s is out of date, run: go generate ./reports/ocsf/...", name)
	}
}

// TestSupportedVersionsHaveSchemas ties the versions the package offers to the
// compiled schemas on disk. The generator generates for whatever is in schemas/,
// so a version listed in only one of the two places would either be selectable
// with no schema behind it or generated for and unreachable.
func TestSupportedVersionsHaveSchemas(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("schemas", "schema-*.json.gz"))
	require.NoError(t, err)

	onDisk := make([]string, 0, len(paths))
	for _, path := range paths {
		name := filepath.Base(path)
		onDisk = append(onDisk, strings.TrimSuffix(strings.TrimPrefix(name, "schema-"), ".json.gz"))
	}
	sort.Strings(onDisk)

	supported := SupportedVersions()
	sort.Strings(supported)

	assert.Equal(t, supported, onDisk,
		"every supported version needs a compiled schema in schemas/, and vice versa")
}
