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
	k8sYaml "sigs.k8s.io/yaml"
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

// ParseYaml loads a yaml file and parse it into the go struct
func ParseYaml(data []byte) (*Bundle, error) {
	// This will generate errors if the yaml contains fields that are not
	// known in the policy.Bundle struct
	err := k8sYaml.UnmarshalStrict(data, &policy.Bundle{})
	if err != nil {
		return nil, err
	}

	baseline := Bundle{}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true) // Enforce strict field mapping

	err = decoder.Decode(&baseline)
	if err != nil {
		return nil, err
	}
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

	err = os.WriteFile(filename, fmtData, 0o644)
	if err != nil {
		return err
	}

	return nil
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
