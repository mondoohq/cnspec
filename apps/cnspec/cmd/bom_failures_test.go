// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cr "go.mondoo.com/mql/cli/reporter"
	"go.mondoo.com/mql/sbom/generator"
)

func TestWithoutFailedAssets(t *testing.T) {
	t.Run("nil report", func(t *testing.T) {
		res, failures := withoutFailedAssets(nil)
		assert.Nil(t, res)
		assert.Empty(t, failures)
	})

	t.Run("no errors keeps the report as is", func(t *testing.T) {
		report := &cr.Report{
			Assets: map[string]*cr.Asset{"//assets/1": {Name: "host-1"}},
			Data:   map[string]*cr.DataValues{"//assets/1": {}},
		}
		res, failures := withoutFailedAssets(report)
		assert.Same(t, report, res)
		assert.Empty(t, failures)
	})

	// An unreachable target (e.g. SSH to a closed port) is recorded as an
	// asset without an MRN or name, an empty data entry and a scan error.
	t.Run("unreachable target", func(t *testing.T) {
		report := &cr.Report{
			Assets: map[string]*cr.Asset{"": {}},
			Data:   map[string]*cr.DataValues{"": {Values: map[string]*cr.DataValue{}}},
			Errors: map[string]string{"": "dial tcp 127.0.0.1:1: connect: connection refused"},
		}
		res, failures := withoutFailedAssets(report)
		assert.Empty(t, res.Assets)
		assert.Empty(t, res.Data)
		assert.Equal(t, []bomFailure{{Asset: "<unnamed asset>", Reason: "dial tcp 127.0.0.1:1: connect: connection refused"}}, failures)
		// no SBOM is generated for it
		assert.Empty(t, generator.GenerateBom(res))
		// the input is untouched
		assert.Len(t, report.Assets, 1)

		err := bomFailuresError("SBOM", failures, 0)
		require.Error(t, err)
		assert.True(t, errors.Is(err, errBomGenerationFailed))
		assert.Contains(t, err.Error(), "could not generate the SBOM for 1 of 1 asset(s)")
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("some assets failed", func(t *testing.T) {
		report := &cr.Report{
			Assets: map[string]*cr.Asset{
				"//assets/ok":   {Name: "ok"},
				"//assets/bad":  {Name: "bad"},
				"//assets/gone": {},
			},
			Data: map[string]*cr.DataValues{
				"//assets/ok":  {},
				"//assets/bad": {},
			},
			Errors: map[string]string{
				"//assets/bad":  "asset doesn't support any policies",
				"//assets/gone": "connection refused",
			},
		}
		res, failures := withoutFailedAssets(report)
		assert.Equal(t, []string{"//assets/ok"}, keys(res.Assets))
		assert.Equal(t, []string{"//assets/ok"}, keys(res.Data))
		assert.Equal(t, []bomFailure{
			{Asset: "//assets/gone", Reason: "connection refused"},
			{Asset: "bad", Reason: "asset doesn't support any policies"},
		}, failures)

		err := bomFailuresError("AIBOM", failures, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not generate the AIBOM for 2 of 3 asset(s)")
		assert.Contains(t, err.Error(), `asset "bad": asset doesn't support any policies`)
		assert.Contains(t, err.Error(), `asset "//assets/gone": connection refused`)
	})
}

func TestBomFailuresError(t *testing.T) {
	assert.NoError(t, bomFailuresError("SBOM", nil, 1))
	assert.NoError(t, bomFailuresError("SBOM", nil, 0))

	err := bomFailuresError("SBOM", []bomFailure{{Asset: "host-1", Reason: "no data points found"}}, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `asset "host-1": no data points found`)
}

func keys[V any](m map[string]V) []string {
	res := make([]string, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	return res
}
