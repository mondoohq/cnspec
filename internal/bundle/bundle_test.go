// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParser(t *testing.T) {
	raw, err := os.ReadFile("../../examples/example.mql.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	baseline, err := ParseYaml(raw)
	require.NoError(t, err)
	assert.NotNil(t, baseline)
	assert.Equal(t, 1, len(baseline.Queries))
	assert.Equal(t, &Impact{
		Value: &ImpactValue{
			Value: 70,
		},
		FileContext: FileContext{79, 13},
	}, baseline.Queries[0].Impact)
}

func TestRemediationDecoding(t *testing.T) {
	t.Run("simple remediation text", func(t *testing.T) {
		desc := "remediation text"
		var r Remediation
		err := yaml.Unmarshal([]byte(desc), &r)
		require.NoError(t, err)
		assert.Equal(t, desc, r.Items[0].Desc)
		assert.Equal(t, "default", r.Items[0].Id)
	})

	t.Run("list of ID + desc", func(t *testing.T) {
		descTyped := `
- id: something
  desc: remediation text
`
		var r Remediation
		err := yaml.Unmarshal([]byte(descTyped), &r)
		require.NoError(t, err)
		assert.Equal(t, "remediation text", r.Items[0].Desc)
		assert.Equal(t, "something", r.Items[0].Id)
	})

	t.Run("items with list of ID + desc (not user-facing!)", func(t *testing.T) {
		descTyped := `
items:
  - id: something
    desc: remediation text
`
		var r Remediation
		err := yaml.Unmarshal([]byte(descTyped), &r)
		require.NoError(t, err)
		assert.Equal(t, "remediation text", r.Items[0].Desc)
		assert.Equal(t, "something", r.Items[0].Id)
	})
}

func TestPreserveHeadComment(t *testing.T) {
	example := `# test
policies:
    - uid: sample-policy
      name: Sample Policy
      version: 1.0.0
`
	var b Bundle
	err := yaml.Unmarshal([]byte(example), &b)
	require.NoError(t, err)

	out, err := yaml.Marshal(&b)
	require.NoError(t, err)
	assert.Equal(t, example, string(out))
}

// Validity is encoded as HumanTime and we check if this is converted properly
func TestParseValidity(t *testing.T) {
	example := `policies:
    - uid: example1
      name: Example policy 1
      groups:
        - filters: asset.family.contains('unix')
          checks:
            - uid: check-05
              title: SSHd should only use very secure ciphers
              impact: 95
              mql: |
                sshd.config.ciphers.all( _ == /ctr/ )
        - type: override
          title: Exception for strong ciphers until September
          valid:
            until: "2025-09-01"
          checks:
            - uid: check-05
              action: preview
`
	var b Bundle
	err := yaml.Unmarshal([]byte(example), &b)
	require.NoError(t, err)

	out, err := yaml.Marshal(&b)
	require.NoError(t, err)
	assert.Equal(t, example, string(out))
}
