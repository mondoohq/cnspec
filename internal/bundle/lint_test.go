package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
)

func TestLint(t *testing.T) {
	rootDir := "./testdata"
	files, err := policy.WalkPolicyBundleFiles(rootDir)
	require.NoError(t, err)

	results, err := Lint(files...)
	require.NoError(t, err)

	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 8, len(results.Entries))
	assert.True(t, results.HasError())

	report, err := results.sarifReport(rootDir)
	require.NoError(t, err)

	assert.Equal(t, 1, len(report.Runs))
	assert.Equal(t, len(rules), len(report.Runs[0].Tool.Driver.Rules))
	assert.Equal(t, 8, len(report.Runs[0].Results))

	data, err := results.ToSarif(rootDir)
	require.NoError(t, err)
	assert.True(t, len(data) > 0)
}
