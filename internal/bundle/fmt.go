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

// ParseYamlWithQueryTitleArray loads a yaml file and parses it, preserving title arrays
func ParseYamlWithQueryTitleArray(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := yaml.Unmarshal(data, &result)
	return result, err
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

	err = os.WriteFile(filename, fmtData, 0o644)
	if err != nil {
		return err
	}

	return nil
}

func FormatFileWithQueryTitleArray(filename string, sort bool) error {
	log.Info().Str("file", filename).Msg("format file with query title arrays")
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Parse as raw YAML to preserve structure
	var rawBundle map[string]interface{}
	err = yaml.Unmarshal(data, &rawBundle)
	if err != nil {
		return err
	}

	// Convert query titles to arrays
	convertQueryTitlesToArraysInMap(rawBundle)

	// If sort is requested, we need to parse into proper structure
	if sort {
		// Marshal back to YAML
		tempData, err := yaml.Marshal(rawBundle)
		if err != nil {
			return err
		}

		// Parse into Bundle for sorting
		b, err := ParseYaml(tempData)
		if err != nil {
			return err
		}

		b.SortContents()

		// Convert back to raw format and ensure titles are arrays
		tempData2, err := FormatBundle(b, false)
		if err != nil {
			return err
		}

		// Parse again and convert titles
		err = yaml.Unmarshal(tempData2, &rawBundle)
		if err != nil {
			return err
		}

		convertQueryTitlesToArraysInMap(rawBundle)
	}

	// Format and write
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err = enc.Encode(rawBundle)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, buf.Bytes(), 0o644)
	if err != nil {
		return err
	}

	return nil
}

// convertQueryTitlesToArraysInMap converts query titles to arrays in a raw YAML map
func convertQueryTitlesToArraysInMap(bundle map[string]interface{}) {
	// Handle queries
	if queries, ok := bundle["queries"]; ok {
		if queryList, ok := queries.([]interface{}); ok {
			for _, q := range queryList {
				if query, ok := q.(map[string]interface{}); ok {
					// Convert title to array if it's a string
					if title, ok := query["title"]; ok {
						switch t := title.(type) {
						case string:
							if t != "" {
								query["title"] = []string{t}
							}
						case []interface{}:
							// Already an array, ensure it's string array
							strArray := make([]string, 0, len(t))
							for _, item := range t {
								if str, ok := item.(string); ok {
									strArray = append(strArray, str)
								}
							}
							if len(strArray) > 0 {
								query["title"] = strArray
							}
						}
					}

					// Handle props within queries
					if props, ok := query["props"]; ok {
						if propList, ok := props.([]interface{}); ok {
							for _, p := range propList {
								if prop, ok := p.(map[string]interface{}); ok {
									if propTitle, ok := prop["title"]; ok {
										switch pt := propTitle.(type) {
										case string:
											if pt != "" {
												prop["title"] = []string{pt}
											}
										case []interface{}:
											// Already an array, ensure it's string array
											strArray := make([]string, 0, len(pt))
											for _, item := range pt {
												if str, ok := item.(string); ok {
													strArray = append(strArray, str)
												}
											}
											if len(strArray) > 0 {
												prop["title"] = strArray
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
