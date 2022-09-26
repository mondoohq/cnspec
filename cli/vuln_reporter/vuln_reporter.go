package vuln_reporter

import (
	"strings"

	"go.mondoo.com/cnquery/resources/packs/core/vadvisor"
)

func RenderReport(report *vadvisor.VulnReport, writer RowWriter, opts RowWriterOpts) error {
	// FIXME: port the vulnerability report renderer
	return nil
}

// filters a leading 0: for rpm versions
func filterZeroEpochPrefix(version string) string {
	return strings.TrimPrefix(version, "0:")
}

func cvesToString(cves []*vadvisor.CVE) []string {
	list := make([]string, len(cves))
	for i := range cves {
		list[i] = cves[i].ID
	}
	return list
}
