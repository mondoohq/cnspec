// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lsp

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"go.mondoo.com/cnquery/v12"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/mqlc"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v12/types"
	"gopkg.in/yaml.v3"
)

// MQLNode represents an MQL code block found in YAML with position tracking
type MQLNode struct {
	Content   string          // The MQL code content
	StartLine int             // Starting line in YAML (0-indexed)
	EndLine   int             // Ending line in YAML (0-indexed)
	StartCol  int             // Starting column in YAML (0-indexed)
	EndCol    int             // Ending column in YAML (0-indexed)
	IsBlock   bool            // Whether this is a block scalar (|) or inline
	Context   *MQLNodeContext // Additional context about this node
	YAMLNode  *yaml.Node      // Reference to the original YAML node
}

// MQLNodeContext provides additional context for an MQL node
type MQLNodeContext struct {
	QueryUID   string              // UID of the query containing this MQL
	QueryMRN   string              // MRN of the query
	PropUID    string              // UID of the prop (if this is a prop MQL)
	PropMRN    string              // MRN of the prop
	IsProp     bool                // Whether this is a prop MQL or main query MQL
	QueryProps map[string]*MQLNode // Props available to this query
}

// YAMLBundle represents a parsed YAML bundle with all MQL nodes extracted
type YAMLBundle struct {
	RootNode    *yaml.Node
	MQLNodes    []*MQLNode
	SourceURI   protocol.DocumentUri
	SourceLines []string // Original source lines for position mapping
}

// YAMLPropsHandler implements mqlc.PropsHandler for YAML bundles
type YAMLPropsHandler struct {
	Props    map[string]*MQLNode       // Map of prop name to MQL node
	Schema   resources.ResourcesSchema // Schema for compiling props
	Features cnquery.Features          // Feature flags
}

// Get retrieves a property by name
func (h *YAMLPropsHandler) Get(name string) *llx.Primitive {
	node, ok := h.Props[name]
	if !ok {
		return nil
	}

	// Try to compile the prop to get its actual type
	// For now, we'll use a simple string type as placeholder
	// In a full implementation, we would cache compiled props
	if node.Content != "" {
		bundle, err := mqlc.Compile(node.Content, nil, mqlc.NewConfig(h.Schema, h.Features))
		if err == nil && bundle != nil && bundle.CodeV2 != nil {
			// Get the return type from the compiled code
			entrypoints := bundle.CodeV2.Entrypoints()
			if len(entrypoints) > 0 {
				chunk := bundle.CodeV2.Chunk(entrypoints[0])
				if chunk != nil {
					resultType := chunk.DereferencedTypeV2(bundle.CodeV2)
					return &llx.Primitive{
						Type: string(resultType),
					}
				}
			}
		}
	}

	// Fallback to Any type if we can't determine the type
	return &llx.Primitive{
		Type: string(types.Any),
	}
}

// Available returns all available props
func (h *YAMLPropsHandler) Available() map[string]*llx.Primitive {
	result := make(map[string]*llx.Primitive, len(h.Props))
	for name := range h.Props {
		result[name] = h.Get(name)
	}
	return result
}

// All returns all possible props
func (h *YAMLPropsHandler) All() map[string]*llx.Primitive {
	return h.Available()
}

// parseYAMLBundle parses a YAML document and extracts all MQL nodes
func (h *MQLHandler) parseYAMLBundle(content string, uri protocol.DocumentUri) (*YAMLBundle, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	bundle := &YAMLBundle{
		RootNode:    &root,
		MQLNodes:    []*MQLNode{},
		SourceURI:   uri,
		SourceLines: strings.Split(content, "\n"),
	}

	// Extract all MQL nodes from the YAML AST
	h.extractMQLNodes(&root, bundle)

	return bundle, nil
}

// extractMQLNodes recursively extracts all MQL nodes from the YAML AST
func (h *MQLHandler) extractMQLNodes(node *yaml.Node, bundle *YAMLBundle) {
	if node == nil {
		return
	}

	// Look for "queries" array at the root
	if node.Kind == yaml.DocumentNode {
		for _, child := range node.Content {
			h.extractMQLNodes(child, bundle)
		}
		return
	}

	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			if keyNode.Value == "queries" && valueNode.Kind == yaml.SequenceNode {
				// Process queries array
				h.extractQueriesArray(valueNode, bundle)
			} else {
				// Continue recursion for other mappings
				h.extractMQLNodes(valueNode, bundle)
			}
		}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			h.extractMQLNodes(item, bundle)
		}
	}
}

// extractQueriesArray processes the queries array and extracts MQL from each query
func (h *MQLHandler) extractQueriesArray(queriesNode *yaml.Node, bundle *YAMLBundle) {
	if queriesNode.Kind != yaml.SequenceNode {
		return
	}

	for _, queryNode := range queriesNode.Content {
		if queryNode.Kind != yaml.MappingNode {
			continue
		}

		h.extractQueryNode(queryNode, bundle)
	}
}

// extractQueryNode extracts MQL from a single query object
func (h *MQLHandler) extractQueryNode(queryNode *yaml.Node, bundle *YAMLBundle) {
	var queryUID, queryMRN string
	var mqlNode *yaml.Node
	var propsNode *yaml.Node

	// Parse query fields
	for i := 0; i < len(queryNode.Content); i += 2 {
		keyNode := queryNode.Content[i]
		valueNode := queryNode.Content[i+1]

		switch keyNode.Value {
		case "uid":
			queryUID = valueNode.Value
		case "mrn":
			queryMRN = valueNode.Value
		case "mql":
			mqlNode = valueNode
		case "props":
			propsNode = valueNode
		}
	}

	// Extract props first so they're available for the main query
	queryProps := make(map[string]*MQLNode)
	if propsNode != nil && propsNode.Kind == yaml.SequenceNode {
		for _, propNode := range propsNode.Content {
			if propNode.Kind != yaml.MappingNode {
				continue
			}

			propMQLNode := h.extractPropNode(propNode, queryUID, queryMRN)
			if propMQLNode != nil {
				queryProps[propMQLNode.Context.PropUID] = propMQLNode
				bundle.MQLNodes = append(bundle.MQLNodes, propMQLNode)
			}
		}
	}

	// Extract main query MQL
	if mqlNode != nil {
		extracted := h.extractMQLFromNode(mqlNode)
		if extracted != nil {
			extracted.Context = &MQLNodeContext{
				QueryUID:   queryUID,
				QueryMRN:   queryMRN,
				IsProp:     false,
				QueryProps: queryProps,
			}
			bundle.MQLNodes = append(bundle.MQLNodes, extracted)
		}
	}
}

// extractPropNode extracts MQL from a single prop object
func (h *MQLHandler) extractPropNode(propNode *yaml.Node, queryUID, queryMRN string) *MQLNode {
	var propUID, propMRN string
	var mqlNode *yaml.Node

	for i := 0; i < len(propNode.Content); i += 2 {
		keyNode := propNode.Content[i]
		valueNode := propNode.Content[i+1]

		switch keyNode.Value {
		case "uid":
			propUID = valueNode.Value
		case "mrn":
			propMRN = valueNode.Value
		case "mql":
			mqlNode = valueNode
		}
	}

	if mqlNode == nil || propUID == "" {
		return nil
	}

	extracted := h.extractMQLFromNode(mqlNode)
	if extracted != nil {
		extracted.Context = &MQLNodeContext{
			QueryUID: queryUID,
			QueryMRN: queryMRN,
			PropUID:  propUID,
			PropMRN:  propMRN,
			IsProp:   true,
		}
	}

	return extracted
}

// extractMQLFromNode extracts MQL content from a YAML node (handles block and inline scalars)
func (h *MQLHandler) extractMQLFromNode(node *yaml.Node) *MQLNode {
	if node == nil || node.Kind != yaml.ScalarNode {
		return nil
	}

	content := node.Value
	if content == "" {
		return nil
	}

	// YAML line numbers are 1-indexed, we convert to 0-indexed
	startLine := node.Line - 1
	endLine := startLine

	// For block scalars (| or >), calculate end line
	isBlock := node.Style == yaml.LiteralStyle || node.Style == yaml.FoldedStyle
	if isBlock {
		lines := strings.Split(content, "\n")
		endLine = startLine + len(lines)
	}

	// YAML Column is 1-indexed and points to where the value starts
	// For inline scalars: points to first character of the value
	// For block scalars: points to the | or > character
	startCol := node.Column - 1

	return &MQLNode{
		Content:   content,
		StartLine: startLine,
		EndLine:   endLine,
		StartCol:  startCol,
		IsBlock:   isBlock,
		YAMLNode:  node,
	}
}

// findMQLNodeAtPosition finds the MQL node that contains the given position
func (h *MQLHandler) findMQLNodeAtPosition(bundle *YAMLBundle, position protocol.Position) *MQLNode {
	line := int(position.Line)

	for _, node := range bundle.MQLNodes {
		// Check if position is within this MQL node
		// For block scalars, the content starts on the line after "mql: |"
		nodeStartLine := node.StartLine
		if node.IsBlock {
			nodeStartLine++ // Skip the "mql: |" line itself
		}

		if line >= nodeStartLine && line <= node.EndLine {
			return node
		}
	}

	return nil
}

// mapYAMLToVirtual maps a position in the YAML file to a position in the virtual MQL document
func (h *MQLHandler) mapYAMLToVirtual(yamlPos protocol.Position, node *MQLNode) protocol.Position {
	return h.mapYAMLToVirtualWithSource(yamlPos, node, nil)
}

// mapYAMLToVirtualWithSource maps with access to source lines for accurate indentation
func (h *MQLHandler) mapYAMLToVirtualWithSource(yamlPos protocol.Position, node *MQLNode, sourceLines []string) protocol.Position {
	yamlLine := int(yamlPos.Line)
	yamlChar := int(yamlPos.Character)

	if !node.IsBlock {
		// Inline scalar: single line
		// node.StartCol points to where the content starts
		relativeChar := yamlChar - node.StartCol
		if relativeChar < 0 {
			relativeChar = 0
		}
		return protocol.Position{
			Line:      0,
			Character: uint32(relativeChar),
		}
	}

	// Block scalar: multi-line
	// Calculate relative line (skip the "mql: |" line)
	relativeLine := yamlLine - node.StartLine - 1
	if relativeLine < 0 {
		relativeLine = 0
	}

	// For block scalars, we need to find the actual indentation from source
	var baseIndent int
	if sourceLines != nil && yamlLine < len(sourceLines) {
		// Find the indentation by looking at the source line
		sourceLine := sourceLines[yamlLine]
		// Count leading spaces
		baseIndent = 0
		for _, ch := range sourceLine {
			if ch == ' ' || ch == '\t' {
				baseIndent++
			} else {
				break
			}
		}
	} else {
		// Fallback: assume content is indented at same column as the | character
		baseIndent = node.StartCol
	}

	relativeChar := yamlChar - baseIndent
	if relativeChar < 0 {
		relativeChar = 0
	}

	return protocol.Position{
		Line:      uint32(relativeLine),
		Character: uint32(relativeChar),
	}
}

// mapVirtualToYAML maps a position in the virtual MQL document back to the YAML file
func (h *MQLHandler) mapVirtualToYAML(virtualPos protocol.Position, node *MQLNode) protocol.Position {
	virtualLine := int(virtualPos.Line)
	virtualChar := int(virtualPos.Character)

	if !node.IsBlock {
		// Inline scalar
		return protocol.Position{
			Line:      uint32(node.StartLine),
			Character: uint32(node.StartCol + virtualChar),
		}
	}

	// Block scalar: add back the base line offset and indentation
	yamlLine := node.StartLine + 1 + virtualLine // +1 to skip "mql: |" line
	baseIndent := node.StartCol + 2
	yamlChar := baseIndent + virtualChar

	return protocol.Position{
		Line:      uint32(yamlLine),
		Character: uint32(yamlChar),
	}
}

// mapRangeVirtualToYAML maps a range from virtual document to YAML
func (h *MQLHandler) mapRangeVirtualToYAML(virtualRange protocol.Range, node *MQLNode) protocol.Range {
	return protocol.Range{
		Start: h.mapVirtualToYAML(virtualRange.Start, node),
		End:   h.mapVirtualToYAML(virtualRange.End, node),
	}
}

// extractPartialMQLForCompletion extracts MQL content up to cursor position for completion
func (h *MQLHandler) extractPartialMQLForCompletion(node *MQLNode, virtualPos protocol.Position) string {
	lines := strings.Split(node.Content, "\n")
	line := int(virtualPos.Line)
	char := int(virtualPos.Character)

	if line >= len(lines) {
		return node.Content
	}

	// Get all lines up to cursor
	var result []string
	for i := 0; i < line; i++ {
		result = append(result, lines[i])
	}

	// Add partial current line
	currentLine := lines[line]
	if char <= len(currentLine) {
		result = append(result, currentLine[:char])
	} else {
		result = append(result, currentLine)
	}

	return strings.Join(result, "\n")
}

// logYAMLBundleInfo logs information about the parsed YAML bundle
func (h *MQLHandler) logYAMLBundleInfo(bundle *YAMLBundle) {
	log.Debug().
		Int("mql_nodes", len(bundle.MQLNodes)).
		Str("uri", string(bundle.SourceURI)).
		Msg("parsed YAML bundle")

	for i, node := range bundle.MQLNodes {
		contextInfo := "unknown"
		if node.Context != nil {
			if node.Context.IsProp {
				contextInfo = fmt.Sprintf("prop %s in query %s", node.Context.PropUID, node.Context.QueryUID)
			} else {
				contextInfo = fmt.Sprintf("query %s (with %d props)", node.Context.QueryUID, len(node.Context.QueryProps))
			}
		}

		log.Debug().
			Int("index", i).
			Str("context", contextInfo).
			Int("start_line", node.StartLine).
			Int("end_line", node.EndLine).
			Bool("is_block", node.IsBlock).
			Int("content_length", len(node.Content)).
			Msg("MQL node")
	}
}
