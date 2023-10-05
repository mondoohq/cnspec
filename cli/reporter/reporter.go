// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"go.mondoo.com/cnquery/v9"
	"go.mondoo.com/cnquery/v9/cli/printer"
	"go.mondoo.com/cnquery/v9/cli/theme/colors"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/mqlc"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v9/shared"
	"go.mondoo.com/cnspec/v9/policy"
	"sigs.k8s.io/yaml"
)

type mqlCode string

const (
	vulnReport      mqlCode = "asset.vulnerabilityReport"
	kernelInstalled mqlCode = "kernel.installed"
)

var _defaultChecksums = map[mqlCode]struct {
	sum string
	err error
}{}

func defaultChecksum(code mqlCode, schema llx.Schema) (string, error) {
	res, ok := _defaultChecksums[code]
	if ok {
		return res.sum, res.err
	}

	codeBundle, err := mqlc.Compile(string(code), nil,
		mqlc.NewConfig(schema, cnquery.DefaultFeatures))
	if err != nil {
		res.err = err
	} else if len(codeBundle.CodeV2.Entrypoints()) != 1 {
		res.err = errors.New("code bundle should only have 1 entrypoint for: " + string(code))
	} else {
		entrypoint := codeBundle.CodeV2.Entrypoints()[0]
		res.sum, ok = codeBundle.CodeV2.Checksums[entrypoint]
		if !ok {
			res.err = errors.New("could not find the datapoint for: " + string(code))
		}
	}

	_defaultChecksums[code] = res
	return res.sum, res.err
}

type Reporter struct {
	Format      Format
	Printer     *printer.Printer
	Colors      *colors.Theme
	IsIncognito bool
	IsVerbose   bool
}

func New(typ string) (*Reporter, error) {
	format, ok := Formats[strings.ToLower(typ)]
	if !ok {
		return nil, errors.New("unknown output format '" + typ + "'. Available: " + AllFormats())
	}

	return &Reporter{
		Format:  format,
		Printer: &printer.DefaultPrinter,
		Colors:  &colors.DefaultColorTheme,
	}, nil
}

func (r *Reporter) Print(data *policy.ReportCollection, out io.Writer) error {
	switch r.Format {
	case Compact:
		rr := &defaultReporter{
			Reporter:  r,
			isCompact: true,
			out:       out,
			data:      data,
		}
		return rr.print()
	case Summary:
		rr := &defaultReporter{
			Reporter:  r,
			isCompact: true,
			isSummary: true,
			out:       out,
			data:      data,
		}
		return rr.print()
	case Full:
		rr := &defaultReporter{
			Reporter:  r,
			isCompact: false,
			out:       out,
			data:      data,
		}
		return rr.print()
	case Report:
		rr := &reportRenderer{
			printer: r.Printer,
			out:     out,
			data:    data,
		}
		return rr.print()
	case YAML:
		raw := bytes.Buffer{}
		writer := shared.IOWriter{Writer: &raw}
		err := ReportCollectionToJSON(data, &writer)
		if err != nil {
			return err
		}

		json, err := yaml.JSONToYAML(raw.Bytes())
		if err != nil {
			return err
		}
		_, err = out.Write(json)
		return err

	case JSON:
		writer := shared.IOWriter{Writer: out}
		return ReportCollectionToJSON(data, &writer)
	case JUnit:
		writer := shared.IOWriter{Writer: out}
		return ReportCollectionToJunit(data, &writer)
	// case CSV:
	// 	res, err = data.ToCsv()
	default:
		return errors.New("unknown reporter type, don't recognize this Format")
	}
}

func (r *Reporter) PrintVulns(data *mvd.VulnReport, out io.Writer, target string) error {
	switch r.Format {
	case Compact:
		rr := &defaultVulnReporter{
			Reporter:  r,
			isCompact: true,
			out:       out,
			data:      data,
			target:    target,
		}
		return rr.print()
	case Summary:
		rr := &defaultVulnReporter{
			Reporter:  r,
			isCompact: true,
			isSummary: true,
			out:       out,
			data:      data,
			target:    target,
		}
		return rr.print()
	case Full:
		rr := &defaultVulnReporter{
			Reporter:  r,
			isCompact: false,
			out:       out,
			data:      data,
			target:    target,
		}
		return rr.print()
	case Report:
		return errors.New("'report' is not supported for vuln reports, please use one of the other formats")
	case JUnit:
		return errors.New("'junit' is not supported for vuln reports, please use one of the other formats")
	case CSV:
		writer := shared.IOWriter{Writer: out}
		return VulnReportToCSV(data, &writer)
	case YAML:
		raw := bytes.Buffer{}
		writer := shared.IOWriter{Writer: &raw}
		err := VulnReportToJSON(target, data, &writer)
		if err != nil {
			return err
		}

		json, err := yaml.JSONToYAML(raw.Bytes())
		if err != nil {
			return err
		}
		_, err = out.Write(json)
		return err
	case JSON:
		writer := shared.IOWriter{Writer: out}
		return VulnReportToJSON(target, data, &writer)
	default:
		return errors.New("unknown reporter type, don't recognize this Format")
	}
}
