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
	indentLevel := 0
	firstQuery := true

	for _, line := range lines {
		// Check if we're entering queries section
		if strings.TrimSpace(line) == "queries:" {
			inQueries = true
			firstQuery = true
			// Determine the indent level of queries
			indentLevel = len(line) - len(strings.TrimLeft(line, " "))
			result = append(result, line)
			continue
		}

		// If we're in queries section
		if inQueries {
			// Check if we're leaving queries section (new top-level key)
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(line, strings.Repeat(" ", indentLevel+2)) &&
				strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(line, strings.Repeat(" ", indentLevel+4)) {
				inQueries = false
				result = append(result, line)
				continue
			}

			// Check if this is the start of a new query (- uid: pattern)
			if strings.HasPrefix(strings.TrimLeft(line, " "), "- uid:") {
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

	// Parse as raw YAML to handle both string and array titles
	var rawBundle map[string]interface{}
	err = yaml.Unmarshal(data, &rawBundle)
	if err != nil {
		return err
	}

	// Normalize all titles to arrays
	normalizeTitlesToArrays(rawBundle)

	// Convert back to YAML for processing
	normalizedData, err := yaml.Marshal(rawBundle)
	if err != nil {
		return err
	}

	// Now parse into Bundle struct (with titles temporarily stored as first array element)
	b, err := parseYamlWithArrayTitles(normalizedData)
	if err != nil {
		return err
	}

	// Format using existing FormatBundle to get proper formatting and sorting
	fmtData, err := FormatBundle(b, sort)
	if err != nil {
		return err
	}

	// Post-process to ensure titles are arrays
	fmtData, err = ensureTitlesAreArrays(fmtData)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, fmtData, 0o644)
	if err != nil {
		return err
	}

	return nil
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
