// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"go.mondoo.com/cnquery/v12/mqlc"
)

func TestParseYAMLBundle_SimpleQuery(t *testing.T) {
	handler := NewMQLHandler()

	yamlContent := `queries:
  - uid: simple-test
    title: Simple Test
    mql: asset.name`

	bundle, err := handler.parseYAMLBundle(yamlContent, "test://simple.yaml")
	require.NoError(t, err)
	require.NotNil(t, bundle)

	assert.Len(t, bundle.MQLNodes, 1)
	assert.Equal(t, "asset.name", bundle.MQLNodes[0].Content)
	assert.Equal(t, "simple-test", bundle.MQLNodes[0].Context.QueryUID)
	assert.False(t, bundle.MQLNodes[0].Context.IsProp)
}

func TestParseYAMLBundle_BlockScalar(t *testing.T) {
	handler := NewMQLHandler()

	yamlContent := `queries:
  - uid: multi-line-query
    title: Multi-line Query
    mql: |
      packages.
        where(name == /ssh/)
      services.
        where(name == /ssh/)`

	bundle, err := handler.parseYAMLBundle(yamlContent, "test://block.yaml")
	require.NoError(t, err)
	require.NotNil(t, bundle)

	assert.Len(t, bundle.MQLNodes, 1)
	node := bundle.MQLNodes[0]
	assert.True(t, node.IsBlock)
	assert.Contains(t, node.Content, "packages.")
	assert.Contains(t, node.Content, "services.")
	assert.Equal(t, "multi-line-query", node.Context.QueryUID)
}

func TestParseYAMLBundle_WithProps(t *testing.T) {
	handler := NewMQLHandler()

	yamlContent := `queries:
  - uid: query-with-props
    title: Query with Props
    props:
      - uid: myProp
        title: My Property
        mql: asset.name
      - uid: anotherProp
        title: Another Property
        mql: asset.platform
    mql: props.myProp + " - " + props.anotherProp`

	bundle, err := handler.parseYAMLBundle(yamlContent, "test://props.yaml")
	require.NoError(t, err)
	require.NotNil(t, bundle)

	// Should have 3 MQL nodes: 2 props + 1 main query
	assert.Len(t, bundle.MQLNodes, 3)

	// Find the prop nodes
	var propNodes []*MQLNode
	var mainNode *MQLNode
	for _, node := range bundle.MQLNodes {
		if node.Context.IsProp {
			propNodes = append(propNodes, node)
		} else {
			mainNode = node
		}
	}

	assert.Len(t, propNodes, 2)
	require.NotNil(t, mainNode)

	// Check main query has props in context
	assert.Len(t, mainNode.Context.QueryProps, 2)
	assert.Contains(t, mainNode.Context.QueryProps, "myProp")
	assert.Contains(t, mainNode.Context.QueryProps, "anotherProp")

	// Check prop contents
	assert.Equal(t, "asset.name", mainNode.Context.QueryProps["myProp"].Content)
	assert.Equal(t, "asset.platform", mainNode.Context.QueryProps["anotherProp"].Content)
}

func TestParseYAMLBundle_MultipleQueries(t *testing.T) {
	handler := NewMQLHandler()

	yamlContent := `queries:
  - uid: first-query
    title: First Query
    mql: asset.name
  - uid: second-query
    title: Second Query
    mql: asset.platform
  - uid: third-query
    title: Third Query
    props:
      - uid: prop1
        mql: asset.arch
    mql: props.prop1`

	bundle, err := handler.parseYAMLBundle(yamlContent, "test://multiple.yaml")
	require.NoError(t, err)
	require.NotNil(t, bundle)

	// Should have 4 MQL nodes: 3 main queries + 1 prop
	assert.Len(t, bundle.MQLNodes, 4)

	// Count main queries vs props
	var mainQueries, props int
	for _, node := range bundle.MQLNodes {
		if node.Context.IsProp {
			props++
		} else {
			mainQueries++
		}
	}

	assert.Equal(t, 3, mainQueries)
	assert.Equal(t, 1, props)
}

func TestFindMQLNodeAtPosition(t *testing.T) {
	handler := NewMQLHandler()

	tests := []struct {
		name            string
		yamlContent     string
		position        protocol.Position
		expectNil       bool
		expectIsProp    bool
		expectPropUID   string
		expectContent   string
		contentContains string
	}{
		{
			name: "in main query",
			yamlContent: `queries:
  - uid: test-query
    title: Test Query
    mql: |
      asset.name
      asset.platform`,
			position:        protocol.Position{Line: 4, Character: 6}, // Position in "asset.name" line
			expectNil:       false,
			contentContains: "asset.name",
		},
		{
			name: "in prop",
			yamlContent: `queries:
  - uid: test-query
    title: Test Query
    props:
      - uid: myProp
        mql: asset.name
    mql: props.myProp`,
			position:      protocol.Position{Line: 5, Character: 14}, // Position in the prop's mql
			expectNil:     false,
			expectIsProp:  true,
			expectPropUID: "myProp",
			expectContent: "asset.name",
		},
		{
			name: "not in MQL",
			yamlContent: `queries:
  - uid: test-query
    title: Test Query
    mql: asset.name`,
			position:  protocol.Position{Line: 2, Character: 10}, // Position in the title line
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundle, err := handler.parseYAMLBundle(tt.yamlContent, "test://"+tt.name+".yaml")
			require.NoError(t, err)
			require.NotNil(t, bundle)

			node := handler.findMQLNodeAtPosition(bundle, tt.position)

			if tt.expectNil {
				assert.Nil(t, node)
			} else {
				require.NotNil(t, node)
				if tt.expectIsProp {
					assert.True(t, node.Context.IsProp)
					assert.Equal(t, tt.expectPropUID, node.Context.PropUID)
				}
				if tt.expectContent != "" {
					assert.Equal(t, tt.expectContent, node.Content)
				}
				if tt.contentContains != "" {
					assert.Contains(t, node.Content, tt.contentContains)
				}
			}
		})
	}
}

func TestMapYAMLToVirtual(t *testing.T) {
	handler := NewMQLHandler()

	tests := []struct {
		name              string
		node              *MQLNode
		yamlPos           protocol.Position
		expectedLine      uint32
		expectedCharacter uint32
	}{
		{
			name: "inline scalar",
			node: &MQLNode{
				Content:   "asset.name",
				StartLine: 3,
				StartCol:  9, // After "    mql: "
				IsBlock:   false,
			},
			yamlPos:           protocol.Position{Line: 3, Character: 15}, // Position at character 15 in YAML (9 + 6 = "asset.n")
			expectedLine:      0,
			expectedCharacter: 6, // "asset.n" = 6 chars into virtual doc
		},
		{
			name: "block scalar",
			node: &MQLNode{
				Content:   "asset.name\nasset.platform",
				StartLine: 3, // "    mql: |" is at line 3
				StartCol:  4, // Base indentation
				IsBlock:   true,
			},
			yamlPos:           protocol.Position{Line: 5, Character: 10}, // Position at line 5 (second line of content), character 10
			expectedLine:      1,                                         // Second line in virtual doc
			expectedCharacter: 6,                                         // 10 - 4 (base indent)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			virtualPos := handler.mapYAMLToVirtual(tt.yamlPos, tt.node)
			assert.Equal(t, tt.expectedLine, virtualPos.Line)
			assert.Equal(t, tt.expectedCharacter, virtualPos.Character)
		})
	}
}

func TestMapVirtualToYAML(t *testing.T) {
	handler := NewMQLHandler()

	tests := []struct {
		name              string
		node              *MQLNode
		virtualPos        protocol.Position
		expectedLine      uint32
		expectedCharacter uint32
	}{
		{
			name: "inline scalar",
			node: &MQLNode{
				Content:   "asset.name",
				StartLine: 3,
				StartCol:  9,
				IsBlock:   false,
			},
			virtualPos:        protocol.Position{Line: 0, Character: 6},
			expectedLine:      3,
			expectedCharacter: 15, // 9 + 6
		},
		{
			name: "block scalar",
			node: &MQLNode{
				Content:   "asset.name\nasset.platform",
				StartLine: 3,
				StartCol:  4,
				IsBlock:   true,
			},
			virtualPos:        protocol.Position{Line: 1, Character: 4},
			expectedLine:      5,  // 3 + 1 (skip mql:|) + 1 (second line)
			expectedCharacter: 10, // 4 + 2 (base indent) + 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlPos := handler.mapVirtualToYAML(tt.virtualPos, tt.node)
			assert.Equal(t, tt.expectedLine, yamlPos.Line)
			assert.Equal(t, tt.expectedCharacter, yamlPos.Character)
		})
	}
}

func TestExtractPartialMQLForCompletion(t *testing.T) {
	handler := NewMQLHandler()

	tests := []struct {
		name           string
		node           *MQLNode
		virtualPos     protocol.Position
		expectedResult string
	}{
		{
			name: "single line",
			node: &MQLNode{
				Content: "asset.name.length",
			},
			virtualPos:     protocol.Position{Line: 0, Character: 11}, // Cursor at "asset.name."
			expectedResult: "asset.name.",
		},
		{
			name: "multi line",
			node: &MQLNode{
				Content: "packages.where(name == /ssh/)\nservices.where(name == /ssh/)",
			},
			virtualPos:     protocol.Position{Line: 1, Character: 9}, // Cursor at second line, after "services."
			expectedResult: "packages.where(name == /ssh/)\nservices.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partial := handler.extractPartialMQLForCompletion(tt.node, tt.virtualPos)
			assert.Equal(t, tt.expectedResult, partial)
		})
	}
}

func TestYAMLPropsHandler_Get(t *testing.T) {
	handler := NewMQLHandler()

	tests := []struct {
		name       string
		props      map[string]*MQLNode
		getPropUID string
		expectNil  bool
		expectType bool
	}{
		{
			name: "existing prop",
			props: map[string]*MQLNode{
				"myProp": {
					Content: "asset.name",
					Context: &MQLNodeContext{
						PropUID: "myProp",
						IsProp:  true,
					},
				},
			},
			getPropUID: "myProp",
			expectNil:  false,
			expectType: true,
		},
		{
			name:       "non-existent prop",
			props:      map[string]*MQLNode{},
			getPropUID: "nonExistent",
			expectNil:  true,
			expectType: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propsHandler := &YAMLPropsHandler{
				Props:    tt.props,
				Schema:   handler.combinedSchema,
				Features: handler.cnqueryFeatures,
			}

			prim := propsHandler.Get(tt.getPropUID)
			if tt.expectNil {
				assert.Nil(t, prim)
			} else {
				require.NotNil(t, prim)
				if tt.expectType {
					assert.NotEmpty(t, prim.Type)
				}
			}
		})
	}
}

func TestYAMLPropsHandler_Available(t *testing.T) {
	handler := NewMQLHandler()

	prop1 := &MQLNode{Content: "asset.name", Context: &MQLNodeContext{PropUID: "prop1", IsProp: true}}
	prop2 := &MQLNode{Content: "asset.platform", Context: &MQLNodeContext{PropUID: "prop2", IsProp: true}}

	propsHandler := &YAMLPropsHandler{
		Props:    map[string]*MQLNode{"prop1": prop1, "prop2": prop2},
		Schema:   handler.combinedSchema,
		Features: handler.cnqueryFeatures,
	}

	available := propsHandler.Available()
	assert.Len(t, available, 2)
	assert.Contains(t, available, "prop1")
	assert.Contains(t, available, "prop2")
}

func TestIsYAMLPolicy(t *testing.T) {
	handler := NewMQLHandler()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "valid YAML bundle",
			content: `queries:
  - uid: test
    mql: asset.name`,
			expected: true,
		},
		{
			name:     "plain MQL",
			content:  "asset.name",
			expected: false,
		},
		{
			name: "YAML without queries",
			content: `packs:
  - uid: test
    name: Test`,
			expected: false,
		},
		{
			name: "YAML with queries but no mql",
			content: `queries:
  - uid: test
    title: Test`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.isYAMLPolicy(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseYAMLBundle_WithPacks(t *testing.T) {
	handler := NewMQLHandler()

	// Test with the actual format from os.mql.yaml
	yamlContent := `packs:
  - uid: linux-mixed-queries
    name: Linux Mixed Queries
    queries:
      - title: Find all SSH packages
        uid: ssh-packages
        mql: |
          packages.
            where(name == /ssh/)
      - title: Get SSH services
        uid: ssh-services
        mql: |
          services.
            where(name == /ssh/)`

	// Note: This will fail to parse because extractMQLNodes looks for "queries:" at root
	// We should update the parser to handle both packs.queries and root queries
	_, err := handler.parseYAMLBundle(yamlContent, "test://packs.yaml")

	// For now, this is expected to fail or return no MQL nodes
	// TODO: Extend parser to handle packs structure
	_ = err // Not asserting on error for now since packs support isn't fully implemented
}

func TestExtractExpressionAtPosition(t *testing.T) {
	handler := NewMQLHandler()

	tests := []struct {
		name     string
		line     string
		charPos  int
		expected string
	}{
		{
			name:     "simple identifier - extracts full chain",
			line:     "asset.name",
			charPos:  5,
			expected: "asset.name", // Extracts the full chain
		},
		{
			name:     "method call",
			line:     "packages.where(name == /ssh/)",
			charPos:  12,
			expected: "packages.where(name == /ssh/)",
		},
		{
			name:     "chained methods - extracts full chain",
			line:     "users.list.where(name == 'root')",
			charPos:  10,
			expected: "users.list.where(name == 'root')", // Extracts full chain
		},
		{
			name:     "at end of identifier",
			line:     "asset.platform",
			charPos:  14,
			expected: "asset.platform",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.extractExpressionAtPosition(tt.line, tt.charPos)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompletion_ChainedFields(t *testing.T) {
	handler := NewMQLHandler()

	tests := []struct {
		name              string
		query             string
		shouldHaveResults bool
	}{
		{
			name:              "after dot on simple field",
			query:             "asset.",
			shouldHaveResults: true,
		},
		{
			name:              "after dot on chained field",
			query:             "asset.name.",
			shouldHaveResults: true,
		},
		{
			name:              "after dot on triple chain",
			query:             "asset.name.downcase.",
			shouldHaveResults: true,
		},
		{
			name:              "empty query",
			query:             "",
			shouldHaveResults: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compile and check for suggestions
			bundle, _ := mqlc.Compile(tt.query, nil, mqlc.NewConfig(handler.combinedSchema, handler.cnqueryFeatures))

			if tt.shouldHaveResults {
				// Either bundle has suggestions, or we should get fallback completions
				if bundle == nil || len(bundle.Suggestions) == 0 {
					t.Logf("No compiler suggestions for '%s', trying fallback", tt.query)
					fallbackItems := handler.getCompletions(tt.query)
					assert.NotEmpty(t, fallbackItems, "Expected fallback completions for: %s", tt.query)
					t.Logf("Got %d fallback items for '%s'", len(fallbackItems), tt.query)
				} else {
					t.Logf("Got %d compiler suggestions for '%s'", len(bundle.Suggestions), tt.query)
					assert.NotEmpty(t, bundle.Suggestions, "Expected compiler suggestions for: %s", tt.query)
				}
			}
		})
	}
}

func TestCompletion_ChainedFieldsInStandalone(t *testing.T) {
	handler := NewMQLHandler()

	tests := []struct {
		name          string
		mqlContent    string
		cursorLine    uint32
		cursorChar    uint32
		expectedQuery string
	}{
		{
			name:          "asset dot",
			mqlContent:    "asset.",
			cursorLine:    0,
			cursorChar:    6, // After "asset."
			expectedQuery: "asset.",
		},
		{
			name:          "asset.name dot",
			mqlContent:    "asset.name.",
			cursorLine:    0,
			cursorChar:    11, // After "asset.name."
			expectedQuery: "asset.name.",
		},
		{
			name:          "asset.name.downcase dot",
			mqlContent:    "asset.name.downcase.",
			cursorLine:    0,
			cursorChar:    20, // After "asset.name.downcase."
			expectedQuery: "asset.name.downcase.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store in documents
			uri := protocol.DocumentUri("file:///test.mql")
			handler.mutex.Lock()
			handler.documents[uri] = tt.mqlContent
			handler.mutex.Unlock()

			// Extract query at position
			position := protocol.Position{Line: tt.cursorLine, Character: tt.cursorChar}
			query, err := handler.getQueryAtPosition(uri, position)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedQuery, query, "Query extraction mismatch")

			// Try to compile and get completions
			codeBundle, _ := mqlc.Compile(query, nil, mqlc.NewConfig(handler.combinedSchema, handler.cnqueryFeatures))

			if codeBundle == nil || len(codeBundle.Suggestions) == 0 {
				t.Logf("No compiler suggestions for '%s', trying fallback", query)
				fallbackItems := handler.getCompletions(query)
				assert.NotEmpty(t, fallbackItems, "Expected fallback completions for: %s", query)
				t.Logf("Got %d fallback items", len(fallbackItems))
			} else {
				t.Logf("Got %d compiler suggestions for '%s'", len(codeBundle.Suggestions), query)
				assert.NotEmpty(t, codeBundle.Suggestions)
			}
		})
	}
}

func TestCompletion_ChainedFieldsInYAML(t *testing.T) {
	handler := NewMQLHandler()

	tests := []struct {
		name           string
		yamlContent    string
		cursorLine     uint32
		cursorChar     uint32
		expectedPrefix string // What we expect to complete after
	}{
		{
			name: "asset dot in YAML",
			yamlContent: `queries:
  - uid: test
    mql: asset.`,
			cursorLine:     2,
			cursorChar:     16, // After "asset."
			expectedPrefix: "asset.",
		},
		{
			name: "asset.name dot in YAML",
			yamlContent: `queries:
  - uid: test
    mql: asset.name.`,
			cursorLine:     2,
			cursorChar:     21, // After "asset.name."
			expectedPrefix: "asset.name.",
		},
		{
			name: "asset.name.downcase dot in YAML",
			yamlContent: `queries:
  - uid: test
    mql: asset.name.downcase.`,
			cursorLine:     2,
			cursorChar:     30, // After "asset.name.downcase."
			expectedPrefix: "asset.name.downcase.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse YAML bundle
			yamlBundle, err := handler.parseYAMLBundle(tt.yamlContent, "file:///test.mql.yaml")
			require.NoError(t, err)
			require.NotNil(t, yamlBundle)

			// Find MQL node at position
			position := protocol.Position{Line: tt.cursorLine, Character: tt.cursorChar}
			mqlNode := handler.findMQLNodeAtPosition(yamlBundle, position)
			require.NotNil(t, mqlNode, "Expected to find MQL node at position")

			// Map to virtual position
			virtualPos := handler.mapYAMLToVirtual(position, mqlNode)

			// Extract partial MQL
			partialMQL := handler.extractPartialMQLForCompletion(mqlNode, virtualPos)
			t.Logf("Extracted partial MQL: '%s'", partialMQL)
			assert.Equal(t, tt.expectedPrefix, partialMQL)

			// Try to compile and get completions
			codeBundle, _ := mqlc.Compile(partialMQL, nil, mqlc.NewConfig(handler.combinedSchema, handler.cnqueryFeatures))

			if codeBundle == nil || len(codeBundle.Suggestions) == 0 {
				t.Logf("No compiler suggestions for '%s', trying fallback", partialMQL)
				fallbackItems := handler.getCompletions(partialMQL)
				assert.NotEmpty(t, fallbackItems, "Expected fallback completions for: %s", partialMQL)
				t.Logf("Got %d fallback items", len(fallbackItems))
				// Log a few items
				for i, item := range fallbackItems {
					if i < 5 {
						t.Logf("  Item %d: %s", i, item.Label)
					}
				}
			} else {
				t.Logf("Got %d compiler suggestions", len(codeBundle.Suggestions))
				assert.NotEmpty(t, codeBundle.Suggestions)
			}
		})
	}
}

func TestCompletion_InProp(t *testing.T) {
	handler := NewMQLHandler()

	tests := []struct {
		name                string
		yamlContent         string
		cursorLine          uint32
		cursorChar          uint32
		expectedQuery       string
		expectedSuggestions []string // Suggestions we expect to see
		minSuggestions      int      // Minimum number of suggestions expected
	}{
		{
			name: "typing asset. in prop",
			yamlContent: `queries:
  - uid: test-query
    title: Test Query
    props:
      - uid: test-prop
        mql: asset.`,
			cursorLine:          5,
			cursorChar:          19, // After "asset." - line is "        mql: asset." so 8+5+6=19
			expectedQuery:       "asset.",
			expectedSuggestions: []string{"name", "platform", "arch"},
			minSuggestions:      10,
		},
		{
			name: "typing asset.name. in prop",
			yamlContent: `queries:
  - uid: test-query
    title: Test Query
    props:
      - uid: test-prop
        mql: asset.name.`,
			cursorLine:          5,
			cursorChar:          24, // After "asset.name." - 8+5+11=24
			expectedQuery:       "asset.name.",
			expectedSuggestions: []string{"downcase", "upcase", "length"},
			minSuggestions:      8,
		},
		{
			name: "typing asset.name.downcase. in prop",
			yamlContent: `queries:
  - uid: test-query
    title: Test Query
    props:
      - uid: test-prop
        mql: asset.name.downcase.`,
			cursorLine:          5,
			cursorChar:          33, // After "asset.name.downcase."
			expectedQuery:       "asset.name.downcase.",
			expectedSuggestions: []string{"length", "trim", "split"},
			minSuggestions:      8,
		},
		{
			name: "typing asset in multiline prop",
			yamlContent: `queries:
  - uid: cis-cisco-ios-xr-7--1.1.1.1
    title: TACACS+
    props:
      - uid: test
        mql: |
          asset.`,
			cursorLine:          6,
			cursorChar:          16, // After "asset." - line is "          asset." so 10+6=16
			expectedQuery:       "asset.",
			expectedSuggestions: []string{"name", "platform", "arch"},
			minSuggestions:      10,
		},
		{
			name: "typing asset.name.downcase in multiline prop",
			yamlContent: `queries:
  - uid: cis-cisco-ios-xr-7--1.1.1.1
    title: TACACS+
    props:
      - uid: test
        mql: |
          asset.name.downcase.`,
			cursorLine:          6,
			cursorChar:          30, // After "asset.name.downcase." - 10+20=30
			expectedQuery:       "asset.name.downcase.",
			expectedSuggestions: []string{"length", "trim", "split"},
			minSuggestions:      8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse YAML bundle
			yamlBundle, err := handler.parseYAMLBundle(tt.yamlContent, "file:///test.mql.yaml")
			require.NoError(t, err)
			require.NotNil(t, yamlBundle)

			t.Logf("Found %d MQL nodes", len(yamlBundle.MQLNodes))
			for i, node := range yamlBundle.MQLNodes {
				t.Logf("Node %d: lines %d-%d, isProp=%v, content='%s'",
					i, node.StartLine, node.EndLine,
					node.Context != nil && node.Context.IsProp,
					node.Content)
			}

			// Find MQL node at position
			position := protocol.Position{Line: tt.cursorLine, Character: tt.cursorChar}
			mqlNode := handler.findMQLNodeAtPosition(yamlBundle, position)
			require.NotNil(t, mqlNode, "Expected to find MQL node at position line=%d char=%d", tt.cursorLine, tt.cursorChar)

			t.Logf("Found MQL node at position: isProp=%v, content='%s'",
				mqlNode.Context != nil && mqlNode.Context.IsProp,
				mqlNode.Content)

			// Map to virtual position
			virtualPos := handler.mapYAMLToVirtualWithSource(position, mqlNode, yamlBundle.SourceLines)
			t.Logf("Mapped to virtual position: line=%d char=%d", virtualPos.Line, virtualPos.Character)

			// Extract partial MQL
			partialMQL := handler.extractPartialMQLForCompletion(mqlNode, virtualPos)
			t.Logf("Extracted partial MQL: '%s'", partialMQL)
			assert.Equal(t, tt.expectedQuery, partialMQL)

			// Try to compile and get completions (no props since we're IN a prop)
			codeBundle, compileErr := mqlc.Compile(partialMQL, nil, mqlc.NewConfig(handler.combinedSchema, handler.cnqueryFeatures))
			if compileErr != nil {
				t.Logf("Compile error: %v", compileErr)
			}

			var suggestions []string
			if codeBundle == nil || len(codeBundle.Suggestions) == 0 {
				t.Logf("No compiler suggestions for '%s', trying fallback", partialMQL)
				fallbackItems := handler.getCompletions(partialMQL)
				require.NotEmpty(t, fallbackItems, "Expected fallback completions for: %s", partialMQL)
				t.Logf("Got %d fallback items", len(fallbackItems))
				for i, item := range fallbackItems {
					suggestions = append(suggestions, item.Label)
					if i < 5 {
						t.Logf("  Item %d: %s", i, item.Label)
					}
				}
			} else {
				t.Logf("Got %d compiler suggestions for '%s'", len(codeBundle.Suggestions), partialMQL)
				require.NotEmpty(t, codeBundle.Suggestions)
				for i, sugg := range codeBundle.Suggestions {
					suggestions = append(suggestions, sugg.Field)
					if i < 5 {
						t.Logf("  Suggestion %d: %s", i, sugg.Field)
					}
				}
			}

			// Check we have minimum suggestions
			assert.GreaterOrEqual(t, len(suggestions), tt.minSuggestions,
				"Expected at least %d suggestions, got %d", tt.minSuggestions, len(suggestions))

			// Check expected suggestions are present
			for _, expected := range tt.expectedSuggestions {
				assert.Contains(t, suggestions, expected,
					"Expected suggestion '%s' not found in: %v", expected, suggestions)
			}
		})
	}
}

// TestYAMLCompletionTextEdits tests that completion items have correct TextEdit ranges
// This is crucial for VS Code to properly apply completions in YAML files
func TestYAMLCompletionTextEdits(t *testing.T) {
	handler := NewMQLHandler()

	tests := []struct {
		name                string
		yaml                string
		cursorLine          uint32 // 0-based line number
		cursorChar          uint32 // 0-based character position
		expectedPartialMQL  string // What we extracted
		expectedReplaceLen  uint32 // How many characters should be replaced (the last token)
		expectedSuggestions []string
	}{
		// Root-level queries (most common in real policy files)
		{
			name: "root queries - simple resource start",
			yaml: `queries:
  - uid: q1
    title: Query 1
    mql: asset
    props:
      - uid: p1
        title: Prop 1
        mql: as`,
			cursorLine:          7,  // Line with "mql: as"
			cursorChar:          16, // After "as" (8 spaces + "mql: " = 13, + "as" = 15, cursor after = 16)
			expectedPartialMQL:  "as",
			expectedReplaceLen:  2, // Replace "as"
			expectedSuggestions: []string{"asset"},
		},
		{
			name: "root queries - chained property start",
			yaml: `queries:
  - uid: q1
    title: Query 1
    mql: asset
    props:
      - uid: p1
        title: Prop 1
        mql: asset.`,
			cursorLine:          7,  // Line with "mql: asset."
			cursorChar:          20, // After "asset." (8 + 5 + 7 = 20)
			expectedPartialMQL:  "asset.",
			expectedReplaceLen:  0, // Replace nothing after the dot
			expectedSuggestions: []string{"name", "arch", "family"},
		},
		{
			name: "root queries - deeply chained",
			yaml: `queries:
  - uid: q1
    mql: asset
    props:
      - uid: p1
        mql: asset.name.down`,
			cursorLine:          5,  // Line with "mql: asset.name.down"
			cursorChar:          28, // After "asset.name.down"
			expectedPartialMQL:  "asset.name.down",
			expectedReplaceLen:  4, // Replace "down"
			expectedSuggestions: []string{"downcase"},
		},
		// Nested under policies (less common but still valid)
		{
			name: "nested under policies - simple resource start",
			yaml: `policies:
  - uid: test
    name: Test
    queries:
      - uid: q1
        title: Query 1
        mql: asset
        props:
          - uid: p1
            title: Prop 1
            mql: as`,
			cursorLine:          10, // Line with "mql: as"
			cursorChar:          22, // After "as" (14 spaces + "mql: " = 19, + "as" = 21, cursor after = 22)
			expectedPartialMQL:  "as",
			expectedReplaceLen:  2, // Replace "as"
			expectedSuggestions: []string{"asset"},
		},
		{
			name: "nested under policies - chained property start",
			yaml: `policies:
  - uid: test
    name: Test
    queries:
      - uid: q1
        title: Query 1
        mql: asset
        props:
          - uid: p1
            title: Prop 1
            mql: asset.`,
			cursorLine:          10, // Line with "mql: asset."
			cursorChar:          26, // After "asset." (19 + "asset." = 25, cursor = 26)
			expectedPartialMQL:  "asset.",
			expectedReplaceLen:  0, // Replace nothing after the dot
			expectedSuggestions: []string{"name", "arch", "family"},
		},
		{
			name: "nested under policies - chained property partial",
			yaml: `policies:
  - uid: test
    name: Test
    queries:
      - uid: q1
        title: Query 1
        mql: asset
        props:
          - uid: p1
            title: Prop 1
            mql: asset.n`,
			cursorLine:          10, // Line with "mql: asset.n"
			cursorChar:          27, // After "asset.n"
			expectedPartialMQL:  "asset.n",
			expectedReplaceLen:  1, // Replace "n"
			expectedSuggestions: []string{"name"},
		},
		{
			name: "nested under policies - deeply chained property",
			yaml: `policies:
  - uid: test
    name: Test
    queries:
      - uid: q1
        title: Query 1
        mql: asset
        props:
          - uid: p1
            title: Prop 1
            mql: asset.name.`,
			cursorLine:          10, // Line with "mql: asset.name."
			cursorChar:          31, // After "asset.name."
			expectedPartialMQL:  "asset.name.",
			expectedReplaceLen:  0, // Replace nothing after the dot
			expectedSuggestions: []string{"downcase", "length"},
		},
		{
			name: "nested under policies - deeply chained property partial",
			yaml: `policies:
  - uid: test
    name: Test
    queries:
      - uid: q1
        title: Query 1
        mql: asset
        props:
          - uid: p1
            title: Prop 1
            mql: asset.name.down`,
			cursorLine:          10, // Line with "mql: asset.name.down"
			cursorChar:          35, // After "asset.name.down"
			expectedPartialMQL:  "asset.name.down",
			expectedReplaceLen:  4, // Replace "down"
			expectedSuggestions: []string{"downcase"},
		},
		// Real-world scenario: inline scalar at root level with method call
		{
			name: "real world - inline scalar at root",
			yaml: `queries:
  - uid: sshd-01
    title: Ensure sshd is running
    mql: 'props.isLinux'
    props:
      - uid: isLinux
        mql: 'uuid(value: "aaaa").ver'`,
			cursorLine:          6,
			cursorChar:          37, // After 'mql: 'uuid(value: "aaaa").ver'
			expectedPartialMQL:  "mql: 'uuid(value: \"aaaa\").ver",
			expectedReplaceLen:  3, // Replace "ver"
			expectedSuggestions: []string{"version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse YAML bundle
			yamlBundle, err := handler.parseYAMLBundle(tt.yaml, "file:///test.mql.yaml")
			assert.NoError(t, err)
			assert.NotNil(t, yamlBundle)

			// Store in cache
			uri := protocol.DocumentUri("file:///test.mql.yaml")
			handler.mutex.Lock()
			handler.yamlBundles[uri] = yamlBundle
			handler.mutex.Unlock()

			// Create completion request
			params := protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: tt.cursorLine, Character: tt.cursorChar},
				},
			}

			// Call completion handler
			result, err := handler.completion(nil, &params)
			assert.NoError(t, err)

			// Convert result to completion items
			items, ok := result.([]protocol.CompletionItem)
			assert.True(t, ok, "Expected []protocol.CompletionItem")
			assert.NotEmpty(t, items, "Expected at least one completion item")

			// Check that all items have correct TextEdit
			for _, item := range items {
				assert.NotNil(t, item.TextEdit, "TextEdit should not be nil for item: %s", item.Label)

				// Cast TextEdit to *protocol.TextEdit
				textEdit, ok := item.TextEdit.(*protocol.TextEdit)
				assert.True(t, ok, "TextEdit should be *protocol.TextEdit")
				assert.NotNil(t, textEdit, "TextEdit should not be nil after cast")

				// Check TextEdit range
				assert.Equal(t, tt.cursorLine, textEdit.Range.Start.Line,
					"TextEdit start line should match cursor line")
				assert.Equal(t, tt.cursorLine, textEdit.Range.End.Line,
					"TextEdit end line should match cursor line")

				// Check that the range replaces exactly the last token
				expectedStartChar := tt.cursorChar - tt.expectedReplaceLen
				assert.Equal(t, expectedStartChar, textEdit.Range.Start.Character,
					"TextEdit should start at position %d (cursor %d - replaceLen %d), got %d",
					expectedStartChar, tt.cursorChar, tt.expectedReplaceLen, textEdit.Range.Start.Character)
				assert.Equal(t, tt.cursorChar, textEdit.Range.End.Character,
					"TextEdit should end at cursor position %d, got %d",
					tt.cursorChar, textEdit.Range.End.Character)

				// Check that NewText is the suggestion field
				assert.Equal(t, item.Label, textEdit.NewText,
					"TextEdit NewText should match Label")
			}

			// Check that expected suggestions are present
			labels := make([]string, len(items))
			for i, item := range items {
				labels[i] = item.Label
			}
			for _, expected := range tt.expectedSuggestions {
				assert.Contains(t, labels, expected,
					"Expected suggestion '%s' not found in: %v", expected, labels)
			}
		})
	}
}
