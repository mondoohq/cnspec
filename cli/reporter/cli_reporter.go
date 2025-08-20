// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12"
	"go.mondoo.com/cnquery/v12/cli/printer"
	"go.mondoo.com/cnquery/v12/cli/theme/colors"
	"go.mondoo.com/cnquery/v12/mqlc"
	"go.mondoo.com/cnquery/v12/providers"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v12/utils/iox"
	"go.mondoo.com/cnspec/v12/policy"
	"sigs.k8s.io/yaml"
)

type mqlCode string

const (
	vulnReportV8    mqlCode = "platform.vulnerabilityReport"
	vulnReportV9    mqlCode = "asset.vulnerabilityReport"
	kernelInstalled mqlCode = "kernel.installed"
)

var _defaultChecksums = map[mqlCode]struct {
	sum string
	err error
}{}

func getVulnReport[T any](results map[string]*T) (*T, error) {
	schema := providers.DefaultRuntime().Schema()
	vulnChecksum, err := defaultChecksum(vulnReportV9, schema)
	if err != nil {
		log.Debug().Err(err).Msg("could not determine vulnerability report checksum")
		return nil, errors.New("no vulnerabilities for this provider")
	}
	if value, ok := results[vulnChecksum]; ok {
		return value, nil
	}

	// FIXME: DEPRECATED, remove in v11.0 vv
	vulnChecksum, err = defaultChecksum(vulnReportV8, schema)
	if err != nil {
		log.Debug().Err(err).Msg("could not determine vulnerability report checksum")
		return nil, errors.New("no vulnerabilities for this provider")
	}
	value, _ := results[vulnChecksum]
	return value, nil
	// ^^
}

func defaultChecksum(code mqlCode, schema resources.ResourcesSchema) (string, error) {
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

// note: implements the OutputHandler interface
type Reporter struct {
	Conf           *PrintConfig
	Printer        *printer.Printer
	Colors         *colors.Theme
	IsIncognito    bool
	ScoreThreshold int
	out            io.Writer
}

func NewReporter(conf *PrintConfig, incognito bool, scoreThreshold int) *Reporter {
	return &Reporter{
		Conf:           conf,
		Printer:        &printer.DefaultPrinter,
		Colors:         &colors.DefaultColorTheme,
		IsIncognito:    incognito,
		ScoreThreshold: scoreThreshold,
		out:            os.Stdout,
	}
}

// This allows to set the output writer directly
func (r *Reporter) WithOutput(out io.Writer) *Reporter {
	r.out = out
	return r
}

func (r *Reporter) WriteReport(ctx context.Context, data *policy.ReportCollection) error {
	features := cnquery.GetFeatures(ctx)
	switch r.Conf.format {
	case FormatCompact, FormatSummary, FormatFull:
		rr := &defaultReporter{
			Reporter:                r,
			output:                  r.out,
			data:                    data,
			isStoreResourcesEnabled: features.IsActive(cnquery.StoreResourcesData),
		}
		return rr.print()
	case FormatReport:
		rr := &reportRenderer{
			printer: r.Printer,
			out:     r.out,
			data:    data,
		}
		return rr.print()
	case FormatYAMLv1:
		yaml, err := reportToYamlV1(data)
		if err != nil {
			return err
		}

		_, err = r.out.Write(yaml)
		return err
	case FormatJSONv1:
		yaml, err := reportToJsonV1(data)
		if err != nil {
			return err
		}

		_, err = r.out.Write(yaml)
		return err
	case FormatJSONv2:
		data, err := reportToJsonV2(data)
		if err != nil {
			return err
		}
		_, err = r.out.Write(data)
		return err
	case FormatYAMLv2:
		data, err := reportToYamlV2(data)
		if err != nil {
			return err
		}
		_, err = r.out.Write(data)
		return err
	case FormatJUnit:
		writer := iox.IOWriter{Writer: r.out}
		return ConvertToJunit(data, &writer)
	// case FormatCSV:
	// 	res, err = data.ToCsv()
	default:
		return errors.New("unknown reporter type, don't recognize this Format")
	}
}

func (r *Reporter) PrintVulns(data *mvd.VulnReport, target string) error {
	if !r.Conf.printVulnerabilities {
		return nil
	}

	switch r.Conf.format {
	case FormatCompact, FormatSummary, FormatFull:
		rr := &defaultVulnReporter{
			Reporter:  r,
			isCompact: r.Conf.isCompact,
			isSummary: !r.Conf.printContents(),
			out:       r.out,
			data:      data,
			target:    target,
		}
		return rr.print()
	case FormatReport:
		return errors.New("'report' is not supported for vuln reports, please use one of the other formats")
	case FormatJUnit:
		return errors.New("'junit' is not supported for vuln reports, please use one of the other formats")
	case FormatCSV:
		writer := iox.IOWriter{Writer: r.out}
		return VulnReportToCSV(data, &writer)
	case FormatYAMLv1, FormatYAMLv2:
		raw := bytes.Buffer{}
		writer := iox.IOWriter{Writer: &raw}
		err := VulnReportToJSON(target, data, &writer)
		if err != nil {
			return err
		}

		json, err := yaml.JSONToYAML(raw.Bytes())
		if err != nil {
			return err
		}
		_, err = r.out.Write(json)
		return err
	case FormatJSONv1, FormatJSONv2:
		writer := iox.IOWriter{Writer: r.out}
		return VulnReportToJSON(target, data, &writer)
	default:
		return errors.New("unknown reporter type, don't recognize this Format")
	}
}
