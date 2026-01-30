// Copyright (c) Mondoo, Inc.
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

func AllFormats() string {
	formats := []string{
		FormatJson, FormatCycloneDxJSON, FormatCycloneDxXML, FormatSpdxJSON, FormatSpdxTagValue, FormatList,
	}

	return strings.Join(formats, ", ")
}

func New(format string) FormatSpecificationHandler {
	switch format {
	case FormatJson, "cnquery-json", "cnspec-json":
		return &CnspecBOM{}
	case FormatCycloneDxJSON:
		return &CycloneDX{
			Format: cyclonedx.BOMFileFormatJSON,
		}
	case FormatCycloneDxXML:
		return &CycloneDX{
			Format: cyclonedx.BOMFileFormatXML,
		}
	case FormatSpdxJSON:
		return &Spdx{
			Version: "2.3",
			Format:  FormatSpdxJSON,
		}
	case FormatSpdxTagValue:
		return &Spdx{
			Version: "2.3",
			Format:  FormatSpdxTagValue,
		}
	case FormatList:
		fallthrough
	default:
		return &TextList{}
	}
}
