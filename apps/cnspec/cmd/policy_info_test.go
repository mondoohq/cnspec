// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyInfoCountsLocalBundleEntries(t *testing.T) {
	const bundle = `policies:
  - uid: demo-profile
    name: Demo profile
    version: 1.0.0
    groups:
      - title: Group A
        filters: asset.family.contains("unix")
        checks:
          - uid: demo-check-1
          - uid: demo-check-2
          - uid: demo-check-3
        queries:
          - uid: demo-query-1
          - uid: demo-query-2
      - title: Group B
        filters: asset.family.contains("unix")
        checks:
          - uid: demo-check-1
          - uid: demo-check-4
          - uid: demo-check-5
        queries:
          - uid: demo-query-1
        policies:
          - uid: child-policy-1
          - uid: child-policy-2
          - uid: child-policy-1
  - uid: child-policy-1
    name: Child policy 1
    version: 1.0.0
  - uid: child-policy-2
    name: Child policy 2
    version: 1.0.0
queries:
  - uid: demo-check-1
    title: Check 1
    mql: "1 == 1"
  - uid: demo-check-2
    title: Check 2
    mql: "2 == 2"
  - uid: demo-check-3
    title: Check 3
    mql: "3 == 3"
  - uid: demo-check-4
    title: Check 4
    mql: "4 == 4"
  - uid: demo-check-5
    title: Check 5
    mql: "5 == 5"
  - uid: demo-query-1
    title: Query 1
    mql: "6 == 6"
  - uid: demo-query-2
    title: Query 2
    mql: "7 == 7"
`

	path := filepath.Join(t.TempDir(), "demo.mql.yaml")
	require.NoError(t, os.WriteFile(path, []byte(bundle), 0o600))
	previousFile := viper.GetString("file")
	viper.Set("file", path)
	t.Cleanup(func() { viper.Set("file", previousFile) })

	read, write, err := os.Pipe()
	require.NoError(t, err)
	previousStdout := os.Stdout
	os.Stdout = write
	defer func() {
		os.Stdout = previousStdout
		read.Close()
		write.Close()
	}()

	err = policyInfoCmd.RunE(policyInfoCmd, []string{"demo-profile"})
	require.NoError(t, err)
	require.NoError(t, write.Close())
	os.Stdout = previousStdout
	output, err := io.ReadAll(read)
	require.NoError(t, err)
	assert.Contains(t, string(output), "Checks:      5\n")
	assert.Contains(t, string(output), "Queries:     2\n")
	assert.Contains(t, string(output), "Policies:    2\n")
}
