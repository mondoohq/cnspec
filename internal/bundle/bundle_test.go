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
		FileContext: FileContext{73, 13},
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
