package reporter

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/theme/colors"
	"go.mondoo.com/cnquery/shared"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/executor"
	"sigs.k8s.io/yaml"
)

var (
	vulnReportDatapointChecksum = executor.MustGetOneDatapoint(executor.MustCompile("platform.vulnerabilityReport"))
	kernelListDatapointChecksum = executor.MustGetOneDatapoint(executor.MustCompile("kernel.installed"))
)

type Reporter struct {
	// Pager set to true will use a pager for the output. Only relevant for all
	// non-json/yaml/junit/csv reports (for now)
	UsePager    bool
	Pager       string
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
			printer:  r.Printer,
			pager:    r.Pager,
			usePager: r.UsePager,
			out:      out,
			data:     data,
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
