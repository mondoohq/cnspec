package reporter

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/theme/colors"
	"go.mondoo.com/cnquery/shared"
	"go.mondoo.com/cnspec/policy"
)

func TestJunitConverter(t *testing.T) {
	reportCollectionRaw, err := os.ReadFile("./testdata/report-ubuntu.json")
	require.NoError(t, err)

	yr := &policy.ReportCollection{}
	err = json.Unmarshal(reportCollectionRaw, yr)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := shared.IOWriter{Writer: &buf}

	r := &Reporter{
		Format:  Formats["compact"],
		Printer: &printer.DefaultPrinter,
		Colors:  &colors.DefaultColorTheme,
	}

	rr := &defaultReporter{
		Reporter:  r,
		isCompact: true,
		out:       &writer,
		data:      yr,
	}
	rr.print()

	assert.Contains(t, buf.String(), "✕ Fail:         Ensure")
	assert.Contains(t, buf.String(), ". Skipped:      Set")
	assert.Contains(t, buf.String(), "! Error:        Set")
	assert.Contains(t, buf.String(), "✓ Pass:  A 100  Ensure")
	assert.Contains(t, buf.String(), "✕ Fail:  F   0  Ensure")
}
