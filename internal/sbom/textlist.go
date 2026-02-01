// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"io"
	"sort"
	"strings"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/v12/cli/theme/colors"
)

var _ Decoder = &TextList{}

// TextList is a simple text list output format
type TextList struct {
	opts renderOpts
}

func (s *TextList) ApplyOptions(opts ...renderOption) {
	for _, opt := range opts {
		opt(&s.opts)
	}
}

func (s *TextList) Convert(bom *Sbom) (any, error) {
	return nil, errConversionNotSupported
}

func (s *TextList) Render(w io.Writer, bom *Sbom) error {
	sort.SliceStable(bom.Packages, func(i, j int) bool {
		if bom.Packages[i].Name != bom.Packages[j].Name {
			return bom.Packages[i].Name < bom.Packages[j].Name
		}

		return bom.Packages[i].Name < bom.Packages[j].Name
	})

	for i := range bom.Packages {
		pkg := bom.Packages[i]

		sb := strings.Builder{}

		// rpm/libxxhash0/0.8.0-2 arm64
		if pkg.Type != "" {
			sb.WriteString(termenv.String(pkg.Type).Foreground(colors.DefaultColorTheme.Secondary).String())
			sb.WriteString(termenv.String("/").Foreground(colors.DefaultColorTheme.Disabled).String())
		}
		sb.WriteString(termenv.String(pkg.Name).Foreground(colors.DefaultColorTheme.Primary).String())
		sb.WriteString(termenv.String("/").Foreground(colors.DefaultColorTheme.Disabled).String())
		sb.WriteString(pkg.Version)

		if pkg.Architecture != "" {
			sb.WriteString(" ")
			sb.WriteString(pkg.Architecture)
		}
		if pkg.Origin != "" {
			sb.WriteString(" (origin:")
			sb.WriteString(pkg.Origin)
			sb.WriteString(")")
		}

		// we only print the location if it is not empty
		// this approach is deprecated and we should remove that once everything moved to evidence
		if pkg.Location != "" {
			sb.WriteString(" ")
			sb.WriteString(termenv.String(pkg.Location).Foreground(colors.DefaultColorTheme.Disabled).String())
		}

		if s.opts.IncludeEvidence {
			for i := range pkg.EvidenceList {
				evidence := pkg.EvidenceList[i]
				sb.WriteString("\n")
				sb.WriteString(termenv.String("  ").Foreground(colors.DefaultColorTheme.Disabled).String())
				sb.WriteString(termenv.String(evidence.Value).Foreground(colors.DefaultColorTheme.Disabled).String())
			}
		}

		sb.WriteString("\n")

		_, err := w.Write([]byte(sb.String()))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *TextList) Parse(r io.ReadSeeker) (*Sbom, error) {
	return nil, errParsingNotSupported
}
