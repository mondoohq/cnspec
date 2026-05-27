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

func AllFormats() string {
	return strings.Join([]string{
		FormatMarkdown, FormatJSON, FormatCycloneDxJSON, FormatCycloneDxXML,
	}, ", ")
}

func NewFormatter(format string) FormatHandler {
	switch format {
	case FormatCycloneDxJSON:
		return &CycloneDXFormatter{Format: cyclonedx.BOMFileFormatJSON}
	case FormatCycloneDxXML:
		return &CycloneDXFormatter{Format: cyclonedx.BOMFileFormatXML}
	case FormatJSON:
		return &JSONFormatter{}
	case FormatMarkdown:
		fallthrough
	default:
		return &TextListFormatter{}
	}
}
