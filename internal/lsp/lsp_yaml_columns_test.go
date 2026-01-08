// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestYAMLNodeColumns(t *testing.T) {
	yamlContent := `queries:
  - uid: test
    mql: asset.name`

	var root yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &root)
	require.NoError(t, err)

	// Navigate to the mql value node
	var findMQL func(*yaml.Node) *yaml.Node
	findMQL = func(n *yaml.Node) *yaml.Node {
		if n.Kind == yaml.ScalarNode && n.Value == "mql" {
			// Return the next node (the value)
			return nil
		}
		for i, child := range n.Content {
			if child.Kind == yaml.ScalarNode && child.Value == "mql" && i+1 < len(n.Content) {
				return n.Content[i+1]
			}
			if result := findMQL(child); result != nil {
				return result
			}
		}
		return nil
	}

	mqlValueNode := findMQL(&root)
	require.NotNil(t, mqlValueNode)

	t.Logf("MQL value node:")
	t.Logf("  Value: '%s'", mqlValueNode.Value)
	t.Logf("  Line: %d (1-indexed)", mqlValueNode.Line)
	t.Logf("  Column: %d (1-indexed)", mqlValueNode.Column)
	t.Logf("  Style: %v", mqlValueNode.Style)

	// Verify: line 2 (0-indexed) is "    mql: asset.name"
	// Column should point to where "asset.name" starts
	// Expected: column 10 (1-indexed) which is position of 'a' in "asset"
	assert.Equal(t, 3, mqlValueNode.Line, "Line should be 3 (1-indexed)")
	assert.Equal(t, 10, mqlValueNode.Column, "Column should be 10 (1-indexed, start of content)")
}

func TestYAMLNodeColumnsBlock(t *testing.T) {
	yamlContent := `queries:
  - uid: test
    mql: |
      asset.name`

	var root yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &root)
	require.NoError(t, err)

	// Navigate to the mql value node
	var findMQL func(*yaml.Node) *yaml.Node
	findMQL = func(n *yaml.Node) *yaml.Node {
		for i, child := range n.Content {
			if child.Kind == yaml.ScalarNode && child.Value == "mql" && i+1 < len(n.Content) {
				return n.Content[i+1]
			}
			if result := findMQL(child); result != nil {
				return result
			}
		}
		return nil
	}

	mqlValueNode := findMQL(&root)
	require.NotNil(t, mqlValueNode)

	t.Logf("MQL value node (block scalar):")
	t.Logf("  Value: '%s'", mqlValueNode.Value)
	t.Logf("  Line: %d (1-indexed)", mqlValueNode.Line)
	t.Logf("  Column: %d (1-indexed)", mqlValueNode.Column)
	t.Logf("  Style: %v (literal=%v)", mqlValueNode.Style, mqlValueNode.Style == yaml.LiteralStyle)

	// For block scalar "mql: |", the Column should point to the | character
	// Line 3 (1-indexed) is "    mql: |"
	// Column should be 10 (1-indexed) which is the position of '|'
	assert.Equal(t, 3, mqlValueNode.Line, "Line should be 3 (1-indexed, the line with 'mql: |')")
	assert.Equal(t, 10, mqlValueNode.Column, "Column should be 10 (1-indexed, position of '|')")
}
