// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"bytes"
	"os"
	"strings"
	"unicode"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v11/policy"
	"gopkg.in/yaml.v3"
)

// Formats the given bundle to a yaml string
func Format(bundle *Bundle) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err := enc.Encode(bundle)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// FormatWithQuerySpacing formats bundle with extra newlines between queries
func FormatWithQuerySpacing(bundle *Bundle) ([]byte, error) {
	data, err := Format(bundle)
	if err != nil {
		return nil, err
	}

	// Add extra newlines between queries
	return addQuerySpacing(data)
}

// addQuerySpacing adds 3 newlines between queries
func addQuerySpacing(data []byte) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	var result []string
	inQueries := false
	queriesIndentLevel := 0
	firstQuery := true

	for _, line := range lines {
		// Check if we're entering queries section
		if strings.TrimSpace(line) == "queries:" {
			inQueries = true
			firstQuery = true
			// Determine the indent level of queries
			queriesIndentLevel = len(line) - len(strings.TrimLeft(line, " "))
			result = append(result, line)
			continue
		}

		// If we're in queries section
		if inQueries {
			// Check if we're leaving queries section (new top-level key at same level as "queries:")
			trimmed := strings.TrimSpace(line)
			currentIndent := len(line) - len(strings.TrimLeft(line, " "))

			if trimmed != "" && currentIndent == queriesIndentLevel && strings.HasSuffix(trimmed, ":") {
				inQueries = false
				result = append(result, line)
				continue
			}

			// Check if this is the start of a new query at the correct indentation level
			// Query items should be at queriesIndentLevel + 2 (for the list indentation)
			expectedQueryIndent := queriesIndentLevel + 2
			if currentIndent == expectedQueryIndent && strings.HasPrefix(strings.TrimLeft(line, " "), "- uid:") {
				if firstQuery {
					// Don't add extra newlines before the first query
					firstQuery = false
				} else {
					// Add extra newlines before subsequent queries
					result = append(result, "", "")
				}
			}
		}

		result = append(result, line)
	}

	return []byte(strings.Join(result, "\n")), nil
}

// FormatRecursive iterates recursively through all .mql.yaml files and formats them
func FormatRecursive(mqlBundlePath string, sort bool) error {
	log.Info().Str("file", mqlBundlePath).Msg("format policy bundle(s)")
	_, err := os.Stat(mqlBundlePath)
	if err != nil {
		return errors.New("file " + mqlBundlePath + " does not exist")
	}

	files, err := policy.WalkPolicyBundleFiles(mqlBundlePath)
	if err != nil {
		return err
	}

	for i := range files {
		f := files[i]
		err := FormatFile(f, sort)
		if err != nil {
			return errors.Wrap(err, "could not format file: "+f)
		}
	}
	return nil
}

// FormatRecursiveWithQueryTitleArray iterates recursively through all .mql.yaml files and formats them
// with query titles as arrays
func FormatRecursiveWithQueryTitleArray(mqlBundlePath string, sort bool) error {
	log.Info().Str("file", mqlBundlePath).Msg("format policy bundle(s) with query title arrays")
	_, err := os.Stat(mqlBundlePath)
	if err != nil {
		return errors.New("file " + mqlBundlePath + " does not exist")
	}

	files, err := policy.WalkPolicyBundleFiles(mqlBundlePath)
	if err != nil {
		return err
	}

	for i := range files {
		f := files[i]
		err := FormatFileWithQueryTitleArray(f, sort)
		if err != nil {
			return errors.Wrap(err, "could not format file: "+f)
		}
	}
	return nil
}

// ParseYaml loads a yaml file and parse it into the go struct
func ParseYaml(data []byte) (*Bundle, error) {
	baseline := Bundle{}

	err := yaml.Unmarshal([]byte(data), &baseline)
	return &baseline, err
}

// sanitizeStringForYaml is here to help generating literal style yaml strings
// if a string has a trailing space in a line, it is automatically converted into quoted style
func sanitizeStringForYaml(s string) string {
	lines := strings.Split(s, "\n")
	for j := range lines {
		content := lines[j]
		content = strings.TrimRightFunc(content, unicode.IsSpace)
		content = strings.ReplaceAll(content, "\t", "  ")
		content = strings.ReplaceAll(content, "\r", "\n")
		// remove all non-printable characters
		content = strings.Map(func(r rune) rune {
			if r == 0x00 {
				return -1
			} else if unicode.IsSpace(r) {
				return r
			} else if unicode.IsGraphic(r) {
				return r
			}
			return -1
		}, content)

		lines[j] = content
	}
	return strings.Join(lines, "\n")
}

func FormatFile(filename string, sort bool) error {
	log.Info().Str("file", filename).Msg("format file")
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	b, err := ParseYaml(data)
	if err != nil {
		return err
	}
	fmtData, err := FormatBundle(b, sort)
	if err != nil {
		return err
	}

	// Add query spacing
	fmtData, err = addQuerySpacing(fmtData)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, fmtData, 0o644)
	if err != nil {
		return err
	}

	return nil
}

// FormatFileWithQueryTitleArray formats a file with query titles as arrays
func FormatFileWithQueryTitleArray(filename string, sort bool) error {
	log.Info().Str("file", filename).Msg("format file with query title arrays")
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// First, parse as raw YAML to extract all titles
	var rawBundle map[string]interface{}
	err = yaml.Unmarshal(data, &rawBundle)
	if err != nil {
		return err
	}

	// Extract and store all titles by UID
	titlesByUID := make(map[string][]string)
	propTitlesByUID := make(map[string]map[string][]string) // uid -> prop_uid -> titles

	if queries, ok := rawBundle["queries"].([]interface{}); ok {
		for _, q := range queries {
			if query, ok := q.(map[string]interface{}); ok {
				uid, hasUID := query["uid"].(string)
				if !hasUID {
					continue
				}

				// Extract all titles for the query
				var titles []string
				switch t := query["title"].(type) {
				case string:
					if t != "" {
						titles = []string{t}
					}
				case []interface{}:
					for _, title := range t {
						if titleStr, ok := title.(string); ok && titleStr != "" {
							titles = append(titles, titleStr)
						}
					}
				}

				if len(titles) > 0 {
					titlesByUID[uid] = titles
				}

				// Extract prop titles
				if props, ok := query["props"].([]interface{}); ok {
					propTitles := make(map[string][]string)
					for _, p := range props {
						if prop, ok := p.(map[string]interface{}); ok {
							propUID, hasPropUID := prop["uid"].(string)
							if !hasPropUID {
								continue
							}

							var propTitleList []string
							switch pt := prop["title"].(type) {
							case string:
								if pt != "" {
									propTitleList = []string{pt}
								}
							case []interface{}:
								for _, title := range pt {
									if titleStr, ok := title.(string); ok && titleStr != "" {
										propTitleList = append(propTitleList, titleStr)
									}
								}
							}

							if len(propTitleList) > 0 {
								propTitles[propUID] = propTitleList
							}
						}
					}
					if len(propTitles) > 0 {
						propTitlesByUID[uid] = propTitles
					}
				}
			}
		}
	}

	// Normalize to single title for Bundle parsing
	normalizeTitlesToSingle(rawBundle)

	// Convert back to YAML for Bundle parsing
	normalizedData, err := yaml.Marshal(rawBundle)
	if err != nil {
		return err
	}

	// Parse into Bundle struct
	b, err := ParseYaml(normalizedData)
	if err != nil {
		return err
	}

	// Format using existing FormatBundle
	fmtData, err := FormatBundle(b, sort)
	if err != nil {
		return err
	}

	// Post-process to restore all titles as arrays
	fmtData, err = restoreTitleArrays(fmtData, titlesByUID, propTitlesByUID)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, fmtData, 0o644)
	if err != nil {
		return err
	}

	return nil
}

// normalizeTitlesToSingle converts array titles to single strings for Bundle parsing
func normalizeTitlesToSingle(data map[string]interface{}) {
	if queries, ok := data["queries"].([]interface{}); ok {
		for _, q := range queries {
			if query, ok := q.(map[string]interface{}); ok {
				// Handle title field
				switch t := query["title"].(type) {
				case []interface{}:
					if len(t) > 0 {
						if titleStr, ok := t[0].(string); ok {
							query["title"] = titleStr
						}
					}
				}

				// Handle props if they exist
				if props, ok := query["props"].([]interface{}); ok {
					for _, p := range props {
						if prop, ok := p.(map[string]interface{}); ok {
							switch pt := prop["title"].(type) {
							case []interface{}:
								if len(pt) > 0 {
									if propTitleStr, ok := pt[0].(string); ok {
										prop["title"] = propTitleStr
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// restoreTitleArrays post-processes formatted YAML to restore title arrays and fix formatting
func restoreTitleArrays(data []byte, titlesByUID map[string][]string, propTitlesByUID map[string]map[string][]string) ([]byte, error) {
	var doc yaml.Node
	err := yaml.Unmarshal(data, &doc)
	if err != nil {
		return nil, err
	}

	// Process the document to restore titles and fix formatting
	restoreTitlesInNode(&doc, titlesByUID, propTitlesByUID)

	// Marshal back to YAML
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err = enc.Encode(&doc)
	if err != nil {
		return nil, err
	}

	result := buf.Bytes()

	// Add query spacing
	result, err = addQuerySpacing(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// restoreTitlesInNode recursively processes YAML nodes to restore title arrays and fix query formatting
func restoreTitlesInNode(node *yaml.Node, titlesByUID map[string][]string, propTitlesByUID map[string]map[string][]string) {
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			restoreTitlesInNode(child, titlesByUID, propTitlesByUID)
		}
	case yaml.MappingNode:
		var currentUID string

		// First pass: find UID if present
		for i := 0; i < len(node.Content)-1; i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]
			if key.Value == "uid" && value.Kind == yaml.ScalarNode {
				currentUID = value.Value
				break
			}
		}

		// Check if this is a query node (has uid and mql/variants)
		isQueryNode := false
		hasUID := false
		hasMql := false
		for i := 0; i < len(node.Content)-1; i += 2 {
			key := node.Content[i]
			if key.Value == "uid" {
				hasUID = true
			} else if key.Value == "mql" || key.Value == "variants" {
				hasMql = true
			}
		}
		isQueryNode = hasUID && hasMql

		// If this is a query node, reorder fields
		if isQueryNode {
			reorderedContent := reorderQueryFields(node.Content, currentUID, titlesByUID, propTitlesByUID)
			node.Content = reorderedContent
		} else {
			// For non-query nodes, process normally
			for i := 0; i < len(node.Content)-1; i += 2 {
				key := node.Content[i]
				value := node.Content[i+1]

				if key.Value == "queries" {
					// Recursively process queries
					restoreTitlesInNode(value, titlesByUID, propTitlesByUID)
				} else {
					// Recursively process other nodes
					restoreTitlesInNode(value, titlesByUID, propTitlesByUID)
				}
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			restoreTitlesInNode(child, titlesByUID, propTitlesByUID)
		}
	}
}

// reorderQueryFields reorders fields in a query node to match the desired format
func reorderQueryFields(content []*yaml.Node, currentUID string, titlesByUID map[string][]string, propTitlesByUID map[string]map[string][]string) []*yaml.Node {
	// Create a map of all fields
	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(content)-1; i += 2 {
		key := content[i]
		value := content[i+1]
		fields[key.Value] = value

		// Fix impact structure if needed
		if key.Value == "impact" && value.Kind == yaml.MappingNode {
			// Check if it's in the nested format { value: X }
			for j := 0; j < len(value.Content)-1; j += 2 {
				impactKey := value.Content[j]
				impactValue := value.Content[j+1]
				if impactKey.Value == "value" {
					// Replace the mapping node with the scalar value
					fields["impact"] = impactValue
					break
				}
			}
		}
	}

	// Restore title arrays if we have them
	if titleValue, ok := fields["title"]; ok && currentUID != "" {
		if titles, ok := titlesByUID[currentUID]; ok && len(titles) > 0 {
			titleArray := &yaml.Node{
				Kind:    yaml.SequenceNode,
				Content: make([]*yaml.Node, 0, len(titles)),
			}
			for _, title := range titles {
				titleArray.Content = append(titleArray.Content, &yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: title,
					Style: titleValue.Style,
				})
			}
			fields["title"] = titleArray
		}
	}

	// Process props if they exist
	if propsValue, ok := fields["props"]; ok && currentUID != "" {
		if propTitles, ok := propTitlesByUID[currentUID]; ok {
			restorePropsInNode(propsValue, propTitles)
		}
		// Also recursively process props
		restoreTitlesInNode(propsValue, titlesByUID, propTitlesByUID)
	}

	// Build reordered content in the desired order
	var reordered []*yaml.Node

	// Define the desired field order:
	// 1. uid
	// 2. title
	// 3. impact
	// 4. filters
	// 5. props
	// 6. mql or variants
	fieldOrder := []string{"uid", "title", "impact", "filters", "props", "mql", "variants"}

	// Add fields in the specified order
	for _, fieldName := range fieldOrder {
		if value, ok := fields[fieldName]; ok {
			reordered = append(reordered, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fieldName,
			}, value)
			delete(fields, fieldName)
		}
	}

	// Add any remaining fields that weren't in our predefined order
	// (like docs, refs, tags, etc.)
	remainingFieldOrder := []string{"docs", "refs", "tags"}
	for _, fieldName := range remainingFieldOrder {
		if value, ok := fields[fieldName]; ok {
			reordered = append(reordered, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fieldName,
			}, value)
			delete(fields, fieldName)
		}
	}

	// Add any other fields that might exist
	for key, value := range fields {
		reordered = append(reordered, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key,
		}, value)
	}

	return reordered
}

// restorePropsInNode processes props to restore their title arrays
func restorePropsInNode(node *yaml.Node, propTitles map[string][]string) {
	if node.Kind != yaml.SequenceNode {
		return
	}

	for _, propNode := range node.Content {
		if propNode.Kind != yaml.MappingNode {
			continue
		}

		var propUID string

		// Find prop UID
		for i := 0; i < len(propNode.Content)-1; i += 2 {
			key := propNode.Content[i]
			value := propNode.Content[i+1]
			if key.Value == "uid" && value.Kind == yaml.ScalarNode {
				propUID = value.Value
				break
			}
		}

		// Restore title if we have it
		if propUID != "" && len(propTitles[propUID]) > 0 {
			for i := 0; i < len(propNode.Content)-1; i += 2 {
				key := propNode.Content[i]
				if key.Value == "title" {
					// Replace with array of titles
					titleArray := &yaml.Node{
						Kind:    yaml.SequenceNode,
						Content: make([]*yaml.Node, 0, len(propTitles[propUID])),
					}
					for _, title := range propTitles[propUID] {
						titleArray.Content = append(titleArray.Content, &yaml.Node{
							Kind:  yaml.ScalarNode,
							Value: title,
							Style: propNode.Content[i+1].Style,
						})
					}
					propNode.Content[i+1] = titleArray
					break
				}
			}
		}
	}
}

// normalizeTitlesToArrays ensures all title fields are arrays in the raw YAML structure
func normalizeTitlesToArrays(data map[string]interface{}) {
	if queries, ok := data["queries"].([]interface{}); ok {
		for _, q := range queries {
			if query, ok := q.(map[string]interface{}); ok {
				// Handle title field
				if title, exists := query["title"]; exists {
					switch t := title.(type) {
					case string:
						if t != "" {
							query["title"] = []interface{}{t}
						}
					case []interface{}:
						// Already an array, keep as is
					default:
						// Handle other cases by converting to string first
						if str, ok := title.(string); ok && str != "" {
							query["title"] = []interface{}{str}
						}
					}
				}

				// Handle props if they exist
				if props, ok := query["props"].([]interface{}); ok {
					for _, p := range props {
						if prop, ok := p.(map[string]interface{}); ok {
							if propTitle, exists := prop["title"]; exists {
								switch pt := propTitle.(type) {
								case string:
									if pt != "" {
										prop["title"] = []interface{}{pt}
									}
								case []interface{}:
									// Already an array, keep as is
								}
							}
						}
					}
				}
			}
		}
	}
}

// parseYamlWithArrayTitles parses YAML where titles are arrays, extracting first element as string
func parseYamlWithArrayTitles(data []byte) (*Bundle, error) {
	// First parse as generic structure
	var raw map[string]interface{}
	err := yaml.Unmarshal(data, &raw)
	if err != nil {
		return nil, err
	}

	// Convert array titles back to strings temporarily for Bundle parsing
	if queries, ok := raw["queries"].([]interface{}); ok {
		for _, q := range queries {
			if query, ok := q.(map[string]interface{}); ok {
				if titleArray, ok := query["title"].([]interface{}); ok && len(titleArray) > 0 {
					if titleStr, ok := titleArray[0].(string); ok {
						query["title"] = titleStr
					}
				}

				// Handle props
				if props, ok := query["props"].([]interface{}); ok {
					for _, p := range props {
						if prop, ok := p.(map[string]interface{}); ok {
							if propTitleArray, ok := prop["title"].([]interface{}); ok && len(propTitleArray) > 0 {
								if propTitleStr, ok := propTitleArray[0].(string); ok {
									prop["title"] = propTitleStr
								}
							}
						}
					}
				}
			}
		}
	}

	// Marshal back and parse as Bundle
	modifiedData, err := yaml.Marshal(raw)
	if err != nil {
		return nil, err
	}

	return ParseYaml(modifiedData)
}

// ensureTitlesAreArrays post-processes formatted YAML to ensure title fields are arrays
func ensureTitlesAreArrays(data []byte) ([]byte, error) {
	var doc yaml.Node
	err := yaml.Unmarshal(data, &doc)
	if err != nil {
		return nil, err
	}

	// Process the document to convert titles
	convertTitlesInNode(&doc)

	// Marshal back to YAML
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err = enc.Encode(&doc)
	if err != nil {
		return nil, err
	}

	result := buf.Bytes()

	// Add query spacing
	result, err = addQuerySpacing(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// convertTitlesInNode recursively processes YAML nodes to convert title fields to arrays
func convertTitlesInNode(node *yaml.Node) {
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			convertTitlesInNode(child)
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content)-1; i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]

			// Convert title fields to arrays
			if key.Value == "title" {
				if value.Kind == yaml.ScalarNode && value.Value != "" {
					// Replace scalar with sequence
					node.Content[i+1] = &yaml.Node{
						Kind: yaml.SequenceNode,
						Content: []*yaml.Node{
							{
								Kind:  yaml.ScalarNode,
								Value: value.Value,
								Style: value.Style,
							},
						},
					}
				}
				// If already a sequence, leave it as is
			} else if key.Value == "queries" || key.Value == "props" {
				// Recursively process queries and props
				convertTitlesInNode(value)
			} else {
				// Recursively process other nodes
				convertTitlesInNode(value)
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			convertTitlesInNode(child)
		}
	}
}

// Format formats the Bundle
func FormatBundle(b *Bundle, sort bool) ([]byte, error) {
	// to improve the formatting we need to remove the whitespace at the end of the lines

	for i := range b.Policies {
		p := b.Policies[i]
		p.Name = sanitizeStringForYaml(p.Name)
		if p.Docs != nil {
			p.Docs.Desc = sanitizeStringForYaml(p.Docs.Desc)
		}
	}

	for i := range b.Queries {
		query := b.Queries[i]
		query.Title = sanitizeStringForYaml(query.Title)
		query.Mql = sanitizeStringForYaml(query.Mql)
		for j := range query.Props {
			query.Props[j].Title = sanitizeStringForYaml(query.Props[j].Title)
			query.Props[j].Mql = sanitizeStringForYaml(query.Props[j].Mql)
			query.Props[j].Desc = sanitizeStringForYaml(query.Props[j].Desc)
		}
		if query.Docs != nil {
			query.Docs.Desc = sanitizeStringForYaml(query.Docs.Desc)
			query.Docs.Audit = sanitizeStringForYaml(query.Docs.Audit)
			if query.Docs.Remediation != nil {
				for j := range query.Docs.Remediation.Items {
					docs := query.Docs.Remediation.Items[j]
					docs.Desc = sanitizeStringForYaml(docs.Desc)
				}
			}
		}
	}

	for i := range b.Frameworks {
		for j := range b.Frameworks[i].Groups {
			grp := b.Frameworks[i].Groups[j]
			grp.Title = sanitizeStringForYaml(grp.Title)
			for k := range grp.Controls {
				grp.Controls[k].Title = sanitizeStringForYaml(grp.Controls[k].Title)
				if grp.Controls[k].Docs != nil {
					grp.Controls[k].Docs.Desc = sanitizeStringForYaml(grp.Controls[k].Docs.Desc)
				}
			}
		}
	}

	if sort {
		b.SortContents()
	}

	return Format(b)
}
