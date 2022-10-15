package reporter

import (
	"errors"
	"io"
	"strings"

	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/theme/colors"
	"go.mondoo.com/cnquery/shared"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/executor"
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
	// case YAML:
	// 	res, err = data.ToYAML()
	case JSON:
		w := shared.IOWriter{Writer: out}
		return ReportCollectionToJSON(data, &w)
	// case JUnit:
	// 	res, err = data.ToJunit()
	// case CSV:
	// 	res, err = data.ToCsv()
	default:
		return errors.New("unknown reporter type, don't recognize this Format")
	}
}
