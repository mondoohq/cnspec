// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"errors"
	"io"
	"strings"

	"github.com/CycloneDX/cyclonedx-go"
)

const (
	FormatJson          string = "json"
	FormatCycloneDxJSON string = "cyclonedx-json"
	FormatCycloneDxXML  string = "cyclonedx-xml"
	FormatSpdxJSON      string = "spdx-json"
	FormatSpdxTagValue  string = "spdx-tag-value"
	FormatList          string = "table"
)

var (
	errConversionNotSupported = errors.New("conversion is not supported")
	errParsingNotSupported    = errors.New("parsing is not supported")
)

type FormatSpecificationHandler interface {
	// Convert converts cnspec sbom to the desired format
	Convert(bom *Sbom) (any, error)
	// Render writes the converted sbom to the writer in the desired format
	Render(w io.Writer, bom *Sbom) error
	// ApplyOptions applies render options to the handler
	ApplyOptions(opts ...renderOption)
	Decoder
}

// format is one value --output accepts: what builds it, and whether it is
// advertised. The undocumented ones are accepted but not listed -- "list" is
// the flag's own default while FormatList is "table", and the two json aliases
// predate this and stay for callers already using them.
//
// AllFormats, IsSupportedFormat and New all read this, so a format cannot be
// constructible but rejected by the guard, or accepted and then silently fall
// through to a default. Those were three hand-kept lists before.
type format struct {
	name       string
	documented bool
	new        func() FormatSpecificationHandler
}

var formats = []format{
	{FormatJson, true, func() FormatSpecificationHandler { return &CnspecBOM{} }},
	{"cnquery-json", false, func() FormatSpecificationHandler { return &CnspecBOM{} }},
	{"cnspec-json", false, func() FormatSpecificationHandler { return &CnspecBOM{} }},
	{FormatCycloneDxJSON, true, func() FormatSpecificationHandler {
		return &CycloneDX{Format: cyclonedx.BOMFileFormatJSON}
	}},
	{FormatCycloneDxXML, true, func() FormatSpecificationHandler {
		return &CycloneDX{Format: cyclonedx.BOMFileFormatXML}
	}},
	{FormatSpdxJSON, true, func() FormatSpecificationHandler {
		return &Spdx{Version: "2.3", Format: FormatSpdxJSON}
	}},
	{FormatSpdxTagValue, true, func() FormatSpecificationHandler {
		return &Spdx{Version: "2.3", Format: FormatSpdxTagValue}
	}},
	{FormatList, true, func() FormatSpecificationHandler { return &TextList{} }},
	{"list", false, func() FormatSpecificationHandler { return &TextList{} }},
}

func lookup(name string) (format, bool) {
	for _, f := range formats {
		if f.name == name {
			return f, true
		}
	}
	return format{}, false
}

// AllFormats lists the documented formats, for the error a bad --output gets.
func AllFormats() string {
	names := make([]string, 0, len(formats))
	for _, f := range formats {
		if f.documented {
			names = append(names, f.name)
		}
	}
	return strings.Join(names, ", ")
}

// IsSupportedFormat reports whether New can render the format. New falls back to
// the table renderer for anything else, so a caller that wants to refuse an
// unknown --output has to ask this first.
func IsSupportedFormat(format string) bool {
	_, ok := lookup(format)
	return ok
}

// New returns the handler for a format. It keeps the historical fallback to the
// table renderer for callers that did not check IsSupportedFormat first.
func New(name string) FormatSpecificationHandler {
	if f, ok := lookup(name); ok {
		return f.new()
	}
	return &TextList{}
}
