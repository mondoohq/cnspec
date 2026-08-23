// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package aibom

//go:generate protoc --plugin=protoc-gen-go=../../scripts/protoc/protoc-gen-go --plugin=protoc-gen-rangerrpc=../../scripts/protoc/protoc-gen-rangerrpc --plugin=protoc-gen-go-vtproto=../../scripts/protoc/protoc-gen-go-vtproto --proto_path=. --go_out=. --go_opt=paths=source_relative --go-vtproto_out=. --go-vtproto_opt=paths=source_relative --go-vtproto_opt=features=marshal+unmarshal+size cnspec_aibom.proto

import (
	"io"
	"strings"

	cyclonedx "github.com/CycloneDX/cyclonedx-go"
)

const (
	FormatCycloneDxJSON string = "cyclonedx-json"
	FormatCycloneDxXML  string = "cyclonedx-xml"
	FormatJSON          string = "json"
	FormatMarkdown      string = "markdown"
)

// FormatHandler renders an AiBom to a specific output format.
type FormatHandler interface {
	Render(w io.Writer, bom *AiBom) error
}

// format is one value --output accepts: what builds it, and whether it is
// advertised. AllFormats, IsSupportedFormat and NewFormatter all read this, so
// a format cannot be constructible but rejected by the guard, or accepted and
// then silently fall through to a default. Those were three hand-kept lists.
type format struct {
	name       string
	documented bool
	new        func() FormatHandler
}

var formats = []format{
	{FormatMarkdown, true, func() FormatHandler { return &TextListFormatter{} }},
	{FormatJSON, true, func() FormatHandler { return &JSONFormatter{} }},
	{FormatCycloneDxJSON, true, func() FormatHandler {
		return &CycloneDXFormatter{Format: cyclonedx.BOMFileFormatJSON}
	}},
	{FormatCycloneDxXML, true, func() FormatHandler {
		return &CycloneDXFormatter{Format: cyclonedx.BOMFileFormatXML}
	}},
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

// IsSupportedFormat reports whether NewFormatter can render the format.
// NewFormatter falls back to markdown for anything else, so a caller that wants
// to refuse an unknown --output has to ask this first.
func IsSupportedFormat(format string) bool {
	_, ok := lookup(format)
	return ok
}

// NewFormatter returns the handler for a format. It keeps the historical
// fallback to markdown for callers that did not check IsSupportedFormat first.
func NewFormatter(name string) FormatHandler {
	if f, ok := lookup(name); ok {
		return f.new()
	}
	return &TextListFormatter{}
}
