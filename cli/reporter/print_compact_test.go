// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
)

func TestBuildCodeIdToMrnMap(t *testing.T) {
	t.Run("nil resolved policy", func(t *testing.T) {
		result := buildCodeIdToMrnMap(nil)
		assert.Nil(t, result)
	})

	t.Run("nil collector job", func(t *testing.T) {
		result := buildCodeIdToMrnMap(&policy.ResolvedPolicy{})
		assert.Nil(t, result)
	})

	t.Run("maps execution query CodeId to parent check MRN", func(t *testing.T) {
		resolved := &policy.ResolvedPolicy{
			CollectorJob: &policy.CollectorJob{
				ReportingJobs: map[string]*policy.ReportingJob{
					"exec-uuid": {
						Uuid:   "exec-uuid",
						QrId:   "codeId123",
						Type:   policy.ReportingJob_EXECUTION_QUERY,
						Notify: []string{"check-uuid"},
					},
					"check-uuid": {
						Uuid: "check-uuid",
						QrId: "//policy.api.mondoo.app/queries/check-mrn",
						Type: policy.ReportingJob_CHECK,
					},
				},
			},
		}

		result := buildCodeIdToMrnMap(resolved)
		require.NotNil(t, result)
		assert.Equal(t, "//policy.api.mondoo.app/queries/check-mrn", result["codeId123"])
	})

	t.Run("skips non-check parents", func(t *testing.T) {
		resolved := &policy.ResolvedPolicy{
			CollectorJob: &policy.CollectorJob{
				ReportingJobs: map[string]*policy.ReportingJob{
					"exec-uuid": {
						Uuid:   "exec-uuid",
						QrId:   "codeId123",
						Type:   policy.ReportingJob_EXECUTION_QUERY,
						Notify: []string{"policy-uuid"},
					},
					"policy-uuid": {
						Uuid: "policy-uuid",
						QrId: "//policy.api.mondoo.app/policies/some-policy",
						Type: policy.ReportingJob_POLICY,
					},
				},
			},
		}

		result := buildCodeIdToMrnMap(resolved)
		require.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("handles CHECK_AND_DATA_QUERY parent", func(t *testing.T) {
		resolved := &policy.ResolvedPolicy{
			CollectorJob: &policy.CollectorJob{
				ReportingJobs: map[string]*policy.ReportingJob{
					"exec-uuid": {
						Uuid:   "exec-uuid",
						QrId:   "codeIdABC",
						Type:   policy.ReportingJob_EXECUTION_QUERY,
						Notify: []string{"both-uuid"},
					},
					"both-uuid": {
						Uuid: "both-uuid",
						QrId: "//policy.api.mondoo.app/queries/both-mrn",
						Type: policy.ReportingJob_CHECK_AND_DATA_QUERY,
					},
				},
			},
		}

		result := buildCodeIdToMrnMap(resolved)
		require.NotNil(t, result)
		assert.Equal(t, "//policy.api.mondoo.app/queries/both-mrn", result["codeIdABC"])
	})

	t.Run("picks first check parent when multiple exist", func(t *testing.T) {
		resolved := &policy.ResolvedPolicy{
			CollectorJob: &policy.CollectorJob{
				ReportingJobs: map[string]*policy.ReportingJob{
					"exec-uuid": {
						Uuid:   "exec-uuid",
						QrId:   "codeId123",
						Type:   policy.ReportingJob_EXECUTION_QUERY,
						Notify: []string{"policy-uuid", "check-uuid"},
					},
					"policy-uuid": {
						Uuid: "policy-uuid",
						QrId: "//policy.api.mondoo.app/policies/some-policy",
						Type: policy.ReportingJob_POLICY,
					},
					"check-uuid": {
						Uuid: "check-uuid",
						QrId: "//policy.api.mondoo.app/queries/check-mrn",
						Type: policy.ReportingJob_CHECK,
					},
				},
			},
		}

		result := buildCodeIdToMrnMap(resolved)
		require.NotNil(t, result)
		assert.Equal(t, "//policy.api.mondoo.app/queries/check-mrn", result["codeId123"])
	})

	t.Run("ignores non-execution-query reporting jobs", func(t *testing.T) {
		resolved := &policy.ResolvedPolicy{
			CollectorJob: &policy.CollectorJob{
				ReportingJobs: map[string]*policy.ReportingJob{
					"check-uuid": {
						Uuid:   "check-uuid",
						QrId:   "//policy.api.mondoo.app/queries/check-mrn",
						Type:   policy.ReportingJob_CHECK,
						Notify: []string{"policy-uuid"},
					},
					"policy-uuid": {
						Uuid: "policy-uuid",
						QrId: "//policy.api.mondoo.app/policies/some-policy",
						Type: policy.ReportingJob_POLICY,
					},
				},
			},
		}

		result := buildCodeIdToMrnMap(resolved)
		require.NotNil(t, result)
		assert.Empty(t, result)
	})
}
