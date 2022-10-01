package components

import (
	"os"

	"go.mondoo.com/cnquery/resources/packs/core/vadvisor"

	"sigs.k8s.io/yaml"
)

// temporary disabled
//func TestRenderAdvisoryResults(t *testing.T) {
//	// load advisory report
//	report, err := loadVulnReportFromFile("./testdata/advisory_report.yaml")
//	require.NoError(t, err)
//	renderer := NewAdvisoryResultTable()
//	output, err := renderer.Render(report)
//	require.NoError(t, err)
//	fmt.Println(output)
//	assert.True(t, len(output) > 0)
//}
//
//func TestRenderEmptyAdvisoryResults(t *testing.T) {
//	// load advisory report
//	report, err := loadVulnReportFromFile("./testdata/advisory_report_empty.yaml")
//	require.NoError(t, err)
//	renderer := NewAdvisoryResultTable()
//	output, err := renderer.Render(report)
//	require.NoError(t, err)
//	fmt.Println(output)
//	assert.True(t, len(output) == 0)
//}

func loadVulnReportFromFile(filename string) (*vadvisor.VulnReport, error) {
	var report vadvisor.VulnReport
	data, err := os.ReadFile(filename)
	if err != nil {
		return &report, err
	}

	err = yaml.Unmarshal(data, &report)
	return &report, err
}
