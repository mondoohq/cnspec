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
	"go.mondoo.com/cnspec/v9/policy"
	"gopkg.in/yaml.v3"
)

// Formats the given bundle to a yaml string
func Format[T any](bundle *T) ([]byte, error) {
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
func FormatRecursive(mqlBundlePath string) error {
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
		err := FormatFile(f)
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
		lines[j] = strings.TrimRightFunc(lines[j], unicode.IsSpace)
	}
	return strings.Join(lines, "\n")
}

// Format formats the .mql.yaml bundle
func FormatFile(filename string) error {
	log.Info().Str("file", filename).Msg("format file")
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	b, err := ParseYaml(data)

	// to improve the formatting we need to remove the whitespace at the end of the lines
	for i := range b.Queries {
		query := b.Queries[i]
		query.Title = sanitizeStringForYaml(query.Title)
		query.Mql = sanitizeStringForYaml(query.Mql)
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

	data, err = Format(b)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, data, 0o644)
	if err != nil {
		return err
	}

	return nil
}

func hasV7Structs(b *Bundle) bool {
	for i := range b.Policies {
		p := b.Policies[i]
		if len(p.Specs) > 0 {
			return true
		}
	}
	return false
}
