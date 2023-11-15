// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mondoo.com/cnquery/v9/explorer"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/testutils"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream"
	"go.mondoo.com/cnspec/v9/policy"
)

func TestFilterPreprocess(t *testing.T) {
	// given
	filters := []string{
		"namespace1/policy1",
		"namespace2/policy2",
		"//registry.mondoo.com/namespace/namespace3/policies/policy3",
	}

	// when
	preprocessed := preprocessPolicyFilters(filters)

	// then
	assert.Equal(t, []string{
		"//registry.mondoo.com/namespace/namespace1/policies/policy1",
		"//registry.mondoo.com/namespace/namespace2/policies/policy2",
		"//registry.mondoo.com/namespace/namespace3/policies/policy3",
	}, preprocessed)
}

func TestGetUpstreamConfig(t *testing.T) {
	t.Run("with job creds", func(t *testing.T) {
		opts := []ScannerOption{
			AllowJobCredentials(),
		}

		pk, err := os.ReadFile("../testdata/private-key.p8")
		require.NoError(t, err)

		cert, err := os.ReadFile("../testdata/cert.pem")
		require.NoError(t, err)

		job := &Job{
			Inventory: &inventory.Inventory{
				Spec: &inventory.InventorySpec{
					UpstreamCredentials: &upstream.ServiceAccountCredentials{
						ApiEndpoint: "api",
						ParentMrn:   "space-mrn",
						PrivateKey:  string(pk),
						Certificate: string(cert),
					},
				},
			},
		}
		scanner := NewLocalScanner(opts...)
		_, err = scanner.getUpstreamConfig(false, job)
		require.NoError(t, err)

		_, err = scanner.getUpstreamConfig(true, &Job{})
		require.NoError(t, err)
	})
}

func TestCreateAssetList(t *testing.T) {
	t.Run("with inventory", func(t *testing.T) {
		job := &Job{
			Inventory: &inventory.Inventory{
				Spec: &inventory.InventorySpec{
					Assets: []*inventory.Asset{
						{
							Connections: []*inventory.Config{
								{
									Type: "k8s",
									Options: map[string]string{
										"path": "./testdata/2pods.yaml",
									},
									Discover: &inventory.Discovery{
										Targets: []string{"auto"},
									},
								},
							},
							ManagedBy: "mondoo-operator-123",
						},
					},
				},
			},
		}
		assetList, candidates, err := createAssetCandidateList(context.TODO(), job, nil, providers.NullRecording{})
		require.NoError(t, err)
		require.Len(t, assetList, 1)
		require.Len(t, candidates, 3)
		require.Equal(t, "mondoo-operator-123", candidates[0].asset.ManagedBy)
		require.Equal(t, "mondoo-operator-123", candidates[1].asset.ManagedBy)
		require.Equal(t, "mondoo-operator-123", candidates[2].asset.ManagedBy)
	})
}

func TestDefaultConfig(t *testing.T) {
	t.Run("without opts", func(t *testing.T) {
		scanner := NewLocalScanner()
		require.NotNil(t, scanner)

		require.Equal(t, providers.NullRecording{}, scanner.recording)
	})
}

type LocalScannerSuite struct {
	suite.Suite
	ctx    context.Context
	schema llx.Schema
	job    *Job
}

func (s *LocalScannerSuite) SetupSuite() {
	s.ctx = context.Background()
	runtime := testutils.Local()
	s.schema = runtime.Schema()
}

func (s *LocalScannerSuite) BeforeTest(suiteName, testName string) {
	s.job = &Job{
		Inventory: &inventory.Inventory{
			Spec: &inventory.InventorySpec{
				Assets: []*inventory.Asset{
					{
						Connections: []*inventory.Config{
							{
								Type: "k8s",
								Options: map[string]string{
									"path": "./testdata/2pods.yaml",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (s *LocalScannerSuite) TestRunIncognito_ExceptionGroups() {
	bundle, err := policy.BundleFromPaths("./testdata/exception-groups.mql.yaml")
	s.Require().NoError(err)

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		Schema:        s.schema,
		RemoveFailing: true,
	})
	s.Require().NoError(err)

	s.job.Bundle = bundle
	s.job.PolicyFilters = []string{"asset-policy"}
	bundleMap := bundle.ToMap()

	ctx := context.Background()
	scanner := NewLocalScanner()
	res, err := scanner.RunIncognito(ctx, s.job)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	full := res.GetFull()
	s.Require().NotNil(full)

	s.Equal(1, len(full.Reports))

	for k, r := range full.Reports {
		// Verify the score is 100
		s.Equal(uint32(100), r.GetScore().Value)

		p := full.ResolvedPolicies[k]

		// Get the code id for all the executed queries
		executedQueries := []string{}
		for qCodeId := range p.ExecutionJob.Queries {
			executedQueries = append(executedQueries, qCodeId)
		}

		expectedQueries := []string{
			bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId,
			bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/sshd-score-01"].CodeId,
		}
		s.ElementsMatch(expectedQueries, executedQueries)

		queryIdToReportingJob := map[string]*policy.ReportingJob{}
		for _, rj := range p.CollectorJob.ReportingJobs {
			_, ok := queryIdToReportingJob[rj.QrId]
			s.Require().False(ok)
			queryIdToReportingJob[rj.QrId] = rj
		}

		// Make sure the ignored query is ignored
		queryRj := queryIdToReportingJob[bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId]
		s.Require().NotNil(queryRj)

		parent := queryRj.Notify[0]
		parentJob := p.CollectorJob.ReportingJobs[parent]
		s.Require().NotNil(parentJob)
		s.Equal(explorer.ScoringSystem_IGNORE_SCORE, parentJob.ChildJobs[queryRj.Uuid].Scoring)
	}
}

func (s *LocalScannerSuite) TestRunIncognito_QueryExceptions() {
	bundle, err := policy.BundleFromPaths("./testdata/exceptions.mql.yaml")
	s.Require().NoError(err)

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		Schema:        s.schema,
		RemoveFailing: true,
	})
	s.Require().NoError(err)

	s.job.Bundle = bundle
	s.job.PolicyFilters = []string{"asset-policy"}
	bundleMap := bundle.ToMap()

	ctx := context.Background()
	scanner := NewLocalScanner()
	res, err := scanner.RunIncognito(ctx, s.job)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	full := res.GetFull()
	s.Require().NotNil(full)

	s.Equal(1, len(full.Reports))

	for k, r := range full.Reports {
		// Verify the score is 100
		s.Equal(uint32(100), r.GetScore().Value)

		p := full.ResolvedPolicies[k]

		// Get the code id for all the executed queries
		executedQueries := []string{}
		for qCodeId := range p.ExecutionJob.Queries {
			executedQueries = append(executedQueries, qCodeId)
		}

		expectedQueries := []string{
			bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId,
			bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/sshd-score-01"].CodeId,
		}
		s.ElementsMatch(expectedQueries, executedQueries)

		queryIdToReportingJob := map[string]*policy.ReportingJob{}
		for _, rj := range p.CollectorJob.ReportingJobs {
			_, ok := queryIdToReportingJob[rj.QrId]
			s.Require().False(ok)
			queryIdToReportingJob[rj.QrId] = rj
		}

		// Make sure the ignored query is ignored
		queryRj := queryIdToReportingJob[bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId]
		s.Require().NotNil(queryRj)

		parent := queryRj.Notify[0]
		parentJob := p.CollectorJob.ReportingJobs[parent]
		s.Require().NotNil(parentJob)
		s.Equal(explorer.ScoringSystem_IGNORE_SCORE, parentJob.ChildJobs[queryRj.Uuid].Scoring)
	}
}

func (s *LocalScannerSuite) TestRunIncognito_QueryExceptions_MultipleGroups() {
	bundle, err := policy.BundleFromPaths("./testdata/exceptions-multiple-groups.mql.yaml")
	s.Require().NoError(err)

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		Schema:        s.schema,
		RemoveFailing: true,
	})
	s.Require().NoError(err)

	s.job.Bundle = bundle
	s.job.PolicyFilters = []string{"asset-policy"}
	bundleMap := bundle.ToMap()

	ctx := context.Background()
	scanner := NewLocalScanner()
	res, err := scanner.RunIncognito(ctx, s.job)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	full := res.GetFull()
	s.Require().NotNil(full)

	s.Equal(1, len(full.Reports))

	for k, r := range full.Reports {
		// Verify the score is 100
		s.Equal(uint32(100), r.GetScore().Value)

		p := full.ResolvedPolicies[k]

		// Get the code id for all the executed queries
		executedQueries := []string{}
		for qCodeId := range p.ExecutionJob.Queries {
			executedQueries = append(executedQueries, qCodeId)
		}

		expectedQueries := []string{
			bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId,
			bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/sshd-score-01"].CodeId,
		}
		s.ElementsMatch(expectedQueries, executedQueries)

		queryIdToReportingJob := map[string]*policy.ReportingJob{}
		for _, rj := range p.CollectorJob.ReportingJobs {
			_, ok := queryIdToReportingJob[rj.QrId]
			s.Require().False(ok)
			queryIdToReportingJob[rj.QrId] = rj
		}

		// Make sure the ignored query is ignored
		queryRj := queryIdToReportingJob[bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId]
		s.Require().NotNil(queryRj)

		parent := queryRj.Notify[0]
		parentJob := p.CollectorJob.ReportingJobs[parent]
		s.Require().NotNil(parentJob)
		s.Equal(explorer.ScoringSystem_IGNORE_SCORE, parentJob.ChildJobs[queryRj.Uuid].Scoring)
	}
}

func TestLocalScannerSuite(t *testing.T) {
	suite.Run(t, new(LocalScannerSuite))
}
