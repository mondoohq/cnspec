package bundle

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	raw, err := os.ReadFile("../../examples/example.mql.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	baseline, err := ParseYaml(raw)
	require.NoError(t, err)
	assert.NotNil(t, baseline)
	assert.Equal(t, 1, len(baseline.Queries))
	assert.Equal(t, &Impact{
		Value: &ImpactValue{
			Value: 70,
		},
		FileContext: FileContext{70, 13},
	}, baseline.Queries[0].Impact)
}

func TestParser_DeprecatedV7(t *testing.T) {
	raw, err := os.ReadFile("../../examples/example.deprecated_v7.mql.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	v8raw, err := DeprecatedV7_ToV8(raw)
	require.NoError(t, err)

	baseline, err := ParseYaml(v8raw)
	require.NoError(t, err)
	assert.NotNil(t, baseline)
	assert.Equal(t, 5, len(baseline.Queries))
	assert.Equal(t, &Impact{
		Value: &ImpactValue{
			Value: 30,
		},
		FileContext: FileContext{27, 13},
	}, baseline.Queries[0].Impact)
}
